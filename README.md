# crd 控制器开发流程
### 环境信息

| 节点   | ip             | 集群版本 | go版本 | 系统版本                      | operator版本 | kustomize版本 |
| ------ | -------------- | -------- | ------ | ----------------------------- | ------------ | ------------- |
| master | 192.168.31.240 | v1.20.0  | 1.15.3 | CentOS Linux release 8.3.2011 | v1.1.0       | v4.1.2        |
| node01 | 192.168.31.209 | v1.20.0  | ---    | CentOS Linux release 8.3.2011 | ---          | ---           |
| node02 | 192.168.31.214 | v1.20.0  | ---    | CentOS Linux release 8.3.2011 | ---          | ---           |

### 搭建集群

参考链接：https://github.com/Mountains-and-rivers/mongo-replica-set

### 在master 节点安装go

```
cd /usr/local/
tar -xvf go1.15.3.linux-amd64.tar.gz
vim /etc/profile
export GOPROXY="https://proxy.golang.org,direct"
export GO111MODULE=on
export GOPATH=/root/go
export PATH=$PATH:/usr/local/go/bin

soource vim /etc/profile
```

### operator-sdk v1.7.0 安装

```
根据client-go 在github 描述
The fastest way to add this library to a project is to run go get k8s.io/client-go@latest with go1.16+. See INSTALL.md for detailed installation instructions and troubleshooting

直接安装go1.16+ 版本，否则client-go模块引用会报错

初始化项目：

operator-sdk init --domain=ydzs.io --license apache2 --owner "Mountains-and-rivers" --repo=github.com/Mountains-and-rivers/opdemo --skip-go-version-check

operator-sdk create api --group app --version v1beta1 --kind MyApp --resource --controller

```

### perator-sdk v1.1.0 安装

```
本次实践采用版本 operator-sdk 1.1.0版本
kustomize v4.1.2 下载
operator v1.1.0 下载

cp kustomize /usr/local/go/bin
chmod +x /usr/local/go/bin/kustomize

cp operator-sdk /usr/local/go/bin
chmod +x /usr/local/go/bin/operator-sdk
```

主要原因如下：

该版本有调试日志，而1.7.0版本没有

![image](https://github.com/Mountains-and-rivers/k8s-crd/blob/main/image/01.png)

### 配置windows远程调试环境

go version 1.16.3  

![image](https://github.com/Mountains-and-rivers/k8s-crd/blob/main/image/02.png)

### 初始化项目

```
export PATH=$PATH:/root/go/bin/
operator-sdk init --domain ydzs.io --license apache2 --owner "Mountains-and-rivers" --repo=github.com/Mountains-and-rivers/opdemo --skip-go-version-check
go mod tidy
make
operator-sdk create api --group app --version v1beta1 --kind MyApp --resource --controller
```



### 控制器开发过程

`app_v1beta1_appservice.yaml`

```
apiVersion: app.ydzs.io/v1beta1
kind: MyApp
metadata:
  name: myapp-demo
spec:
  size: 2
  image: nginx:1.19.1
  ports:
    - port: 80
      targetPort: 80
      nodePort: 30002
```

定义期望状态

D:\golang\src\k8s.io\opdemo\api\v1beta1\myapp_types.go

```
// MyAppSpec defines the desired state of MyApp
type MyAppSpec struct {
	Size *int32 `json:"size"`
	Image string `json:"image"`
	Ports []corev1.ServicePort `json:"ports"`
	Resources corev1.ResourceRequirements `json:"resource,omitempty"`
	Envs []corev1.EnvVar `json:"envs,omitempty"`
}

// MyAppStatus defines the observed state of MyApp
type MyAppStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	appsv1.DeploymentStatus `json:",inline"`
}



修改完重新make 一下
```

添加业务逻辑

D:\golang\src\k8s.io\opdemo\controllers\myapp_controller.go

```
/*
Copyright 2021 Mountains-and-rivers.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	appv1beta1 "github.com/Mountains-and-rivers/opdemo/api/v1beta1"
)

var (
	oldSpecAnnotation = "old/spec"
)
// MyAppReconciler reconciles a MyApp object
type MyAppReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=app.ydzs.io,resources=myapps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=app.ydzs.io,resources=myapps/status,verbs=get;update;patch

func (r *MyAppReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("myapp", req.NamespacedName)

	// 首先获取MyApp 实例
	var myapp appv1beta1.MyApp
	if err := r.Client.Get(ctx,req.NamespacedName,&myapp); err != nil {
         // MyApp was deleted, Ignore
		return ctrl.Result{},client.IgnoreNotFound(err)
	}


	// delete MyApp 过后去创建对应的Deployment 和 Service(观察当前状态和期望状态进行对比)
	// 创建就得去判断是否存在，存在就忽略，不存在就创建

	//调谐，获取到当前的一个状态，然后和我们期望的状态进行比对
	// CreateOrUpdate Deployment
	var deploy appsv1.Deployment
	deploy.Name = myapp.Name
	deploy.Namespace = myapp.Namespace
	or, err := ctrl.CreateOrUpdate(ctx,r,&deploy, func() error {
		//调谐必须在函数中实现
		Muatedeployment(&myapp,&deploy)
		return  controllerutil.SetControllerReference(&myapp,&deploy,r.Scheme)
	})
	if err != nil {
		return ctrl.Result{}, err
	}
	log.Info("CreateOrUpdate","Deployment",or)

	//CreateOrUpdate Service
	var service corev1.Service
	service.Name = myapp.Name
	service.Namespace = myapp.Namespace
	or, err = ctrl.CreateOrUpdate(ctx,r,&service, func() error {
		//调谐必须在函数中实现
		MuateService(&myapp,&service)
		return  controllerutil.SetControllerReference(&myapp,&service,r.Scheme)
	})
	if err != nil {
		return ctrl.Result{}, err
	}
	log.Info("CreateOrUpdate","Service",or)
	return ctrl.Result{}, nil
}


func (r *MyAppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		For(&appv1beta1.MyApp{}).
		Complete(r)
}
```

D:\golang\src\k8s.io\opdemo\controllers\resource.go

```
package controllers

import (
	"github.com/Mountains-and-rivers/opdemo/api/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func Muatedeployment(app *v1beta1.MyApp,deploy  *appsv1.Deployment)  {
	labels := map[string]string{"myapp": app.Name}
	selector := &metav1.LabelSelector{
		MatchLabels: labels,
	}
	deploy.Spec = appsv1.DeploymentSpec{
		Replicas: app.Spec.Size,
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: labels,
			},
			Spec: corev1.PodSpec{
				Containers: newContainers(app),
			},
		},
		Selector: selector,
	}
}

func MuateService(app *v1beta1.MyApp,service *corev1.Service )  {
	service.Spec = corev1.ServiceSpec{
		ClusterIP: service.Spec.ClusterIP,
		Ports: app.Spec.Ports,
		Type: corev1.ServiceTypeNodePort,
		Selector: map[string]string{
			"myapp": app.Name,
		},
	}
}

func NewDeploy(app *v1beta1.MyApp) *appsv1.Deployment {
	labels := map[string]string{"myapp": app.Name}
	selector := &metav1.LabelSelector{
		MatchLabels: labels,
	}
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: app.Namespace,
			OwnerReferences: makeOwnerReferences(app),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: app.Spec.Size,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: newContainers(app),
				},
			},
			Selector: selector,
		},
	}
}

func makeOwnerReferences(app *v1beta1.MyApp)  []metav1.OwnerReference{
	return 	[]metav1.OwnerReference{
		*metav1.NewControllerRef(app, schema.GroupVersionKind{
			Kind:    v1beta1.Kind,
			Group:   v1beta1.GroupVersion.Group,
			Version: v1beta1.GroupVersion.Version,
		}),
	}
}

func newContainers(app *v1beta1.MyApp) []corev1.Container {
	containerPorts := []corev1.ContainerPort{}
	for _, svcPort := range app.Spec.Ports {
		containerPorts = append(containerPorts, corev1.ContainerPort{
			ContainerPort: svcPort.TargetPort.IntVal,
		})
	}
	return []corev1.Container{
		{
			Name:      app.Name,
			Image:     app.Spec.Image,
			Resources: app.Spec.Resources,
			Env:       app.Spec.Envs,
			Ports:     containerPorts,
		},
	}
}

func NewService(app *v1beta1.MyApp) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind: "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: app.Namespace,
			OwnerReferences: makeOwnerReferences(app),
		},
		Spec: corev1.ServiceSpec{
			Ports: app.Spec.Ports,
			Type: corev1.ServiceTypeNodePort,
			Selector: map[string]string{
				"myapp": app.Name,
			},
		},
	}
}
```

调试代码

```
make install #安装crd
make run ENABLE_WEBHOOK=false # 启动控制器

通过对资源对象进行各种操作可以看到日志输出变化
```

![image](https://github.com/Mountains-and-rivers/k8s-crd/blob/main/image/03.png)


### 部署发布

镜像制作

````

[root@node opdemo]#  docker login

登录仓库

[root@node opdemo]# export USERNAME=mangseng
[root@node opdemo]# make docker-build IMG=$USERNAME/opdemo:v1.0.0
[root@node opdemo]# make docker-push IMG=$USERNAME/opdemo:v1.0.0
/root/go/bin/controller-gen object:headerFile="hack/boilerplate.go.txt" paths="./..."
go fmt ./...
go vet ./...

/root/go/bin/controller-gen "crd:trivialVersions=true" rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases
mkdir -p /root/go/src/opdemo/testbin
test -f /root/go/src/opdemo/testbin/setup-envtest.sh || curl -sSLo /root/go/src/opdemo/testbin/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.6.3/hack/setup-envtest.sh
source /root/go/src/opdemo/testbin/setup-envtest.sh; fetch_envtest_tools /root/go/src/opdemo/testbin; setup_envtest_env /root/go/src/opdemo/testbin; go test ./... -coverprofile cover.out
Using cached envtest tools from /root/go/src/opdemo/testbin
setting up env vars
?   	github.com/Mountains-and-rivers/opdemo	[no test files]
?   	github.com/Mountains-and-rivers/opdemo/api/v1beta1	[no test files]
ok  	github.com/Mountains-and-rivers/opdemo/controllers	5.943s	coverage: 0.0% of statements
docker build . -t mangseng/opdemo:v1.0.0
Sending build context to Docker daemon  283.4MB
Step 1/14 : FROM golang:1.13 as builder
1.13: Pulling from library/golang
d6ff36c9ec48: Pull complete 
c958d65b3090: Pull complete 
edaf0a6b092f: Pull complete 
80931cf68816: Pull complete 
813643441356: Pull complete 
799f41bb59c9: Pull complete 
16b5038bccc8: Pull complete 
Digest: sha256:8ebb6d5a48deef738381b56b1d4cd33d99a5d608e0d03c5fe8dfa3f68d41a1f8
Status: Downloaded newer image for golang:1.13
 ---> d6f3656320fe
Step 2/14 : WORKDIR /workspace
 ---> Running in d5584cc1a05e
Removing intermediate container d5584cc1a05e
 ---> b4275eb38007
Step 3/14 : COPY go.mod go.mod
 ---> 6bc3ecdfd7b8
Step 4/14 : COPY go.sum go.sum
 ---> 670ba33bc027
Step 5/14 : RUN go mod download
 ---> Running in 793b093691ac
...
...

Removing intermediate container 793b093691ac
 ---> 22baf43e3997
Step 6/14 : COPY main.go main.go
 ---> c8eab00640f1
Step 7/14 : COPY api/ api/
 ---> d69d1b602fa7
Step 8/14 : COPY controllers/ controllers/
 ---> ca0778a065c9
Step 9/14 : RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o manager main.go
 ---> Running in 69a831b34415
Removing intermediate container 69a831b34415
 ---> 43d1500c29a0
Step 10/14 : FROM gcr.io/distroless/static:nonroot
nonroot: Pulling from distroless/static
5dea5ec2316d: Pull complete 
Digest: sha256:cd784033c94dd30546456f35de8e128390ae15c48cbee5eb7e3306857ec17631
Status: Downloaded newer image for gcr.io/distroless/static:nonroot
 ---> fb7b4da47366
Step 11/14 : WORKDIR /
 ---> Running in 8237732ffffb
Removing intermediate container 8237732ffffb
 ---> 8e06bf55e177
Step 12/14 : COPY --from=builder /workspace/manager .
 ---> 85f2a5ad9258
Step 13/14 : USER nonroot:nonroot
 ---> Running in 8584b4ffd4d2
Removing intermediate container 8584b4ffd4d2
 ---> c48afce10ead
Step 14/14 : ENTRYPOINT ["/manager"]
 ---> Running in 69c1b43e72c6
Removing intermediate container 69c1b43e72c6
 ---> 02f2dfd16f6d
Successfully built 02f2dfd16f6d
Successfully tagged mangseng/opdemo:v1.0.0
[root@node opdemo]# make docker-push IMG=$USERNAME/opdemo:v1.0.0
docker push mangseng/opdemo:v1.0.0
The push refers to repository [docker.io/mangseng/opdemo]
37193d79c55e: Pushed 
417cb9b79ade: Pushed 
v1.0.0: digest: sha256:fd203ba194b53be9adaab041a7f160991599f11349056a768ce2dad6d7572582 size: 739
````

![image](https://github.com/Mountains-and-rivers/k8s-crd/blob/main/image/04.png)

make 控制器部署

```
[root@node opdemo]# make deploy IMG=$USERNAME/opdemo:v1.0.0

/root/go/bin/controller-gen "crd:trivialVersions=true" rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases
cd config/manager && /usr/local/bin/kustomize edit set image controller=mangseng/opdemo:v1.0.0
/usr/local/bin/kustomize build config/default | kubectl apply -f -
namespace/opdemo-system created
Warning: apiextensions.k8s.io/v1beta1 CustomResourceDefinition is deprecated in v1.16+, unavailable in v1.22+; use apiextensions.k8s.io/v1 CustomResourceDefinition
customresourcedefinition.apiextensions.k8s.io/myapps.app.ydzs.io configured
role.rbac.authorization.k8s.io/opdemo-leader-election-role created
clusterrole.rbac.authorization.k8s.io/opdemo-manager-role created
clusterrole.rbac.authorization.k8s.io/opdemo-proxy-role created
Warning: rbac.authorization.k8s.io/v1beta1 ClusterRole is deprecated in v1.17+, unavailable in v1.22+; use rbac.authorization.k8s.io/v1 ClusterRole
clusterrole.rbac.authorization.k8s.io/opdemo-metrics-reader created
rolebinding.rbac.authorization.k8s.io/opdemo-leader-election-rolebinding created
clusterrolebinding.rbac.authorization.k8s.io/opdemo-manager-rolebinding created
clusterrolebinding.rbac.authorization.k8s.io/opdemo-proxy-rolebinding created
service/opdemo-controller-manager-metrics-service created
deployment.apps/opdemo-controller-manager created

```
由于我的网络没有阻碍，可以拉取各种镜像，如果遇到镜像拉取失败，可以从这里拉取镜像，然后修改tag，push到docker官方仓库  

https://www.katacoda.com/

注册登录  

![image](https://github.com/Mountains-and-rivers/k8s-crd/blob/main/image/05.png)

crd部署

```
 # 和上面内容修改一致

 kubectl apply -f config/samples/app_v1beta1_appservice.yaml
```

### 验证  

```
[root@node opdemo]# kubectl get pod -n opdemo-system
NAME                                         READY   STATUS    RESTARTS   AGE
opdemo-controller-manager-57bccc448f-v7blh   2/2     Running   0          20m

[root@node opdemo]# kubectl get pod -n opdemo-system
NAME                                         READY   STATUS    RESTARTS   AGE
opdemo-controller-manager-57bccc448f-v7blh   2/2     Running   0          20m

[root@node opdemo]# kubectl logs -f opdemo-controller-manager-57bccc448f-v7blh -n opdemo-system
error: a container name must be specified for pod opdemo-controller-manager-57bccc448f-v7blh, choose one of: [kube-rbac-proxy manager]
[root@node opdemo]# kubectl logs -f opdemo-controller-manager-57bccc448f-v7blh -n opdemo-system manager
2021-05-02T04:04:12.827Z	INFO	controller-runtime.metrics	metrics server is starting to listen	{"addr": "127.0.0.1:8080"}
2021-05-02T04:04:12.828Z	INFO	setup	starting manager
I0502 04:04:12.828899       1 leaderelection.go:242] attempting to acquire leader lease  opdemo-system/04d31993.ydzs.io...
2021-05-02T04:04:12.828Z	INFO	controller-runtime.manager	starting metrics server	{"path": "/metrics"}
I0502 04:04:12.838590       1 leaderelection.go:252] successfully acquired lease opdemo-system/04d31993.ydzs.io
2021-05-02T04:04:12.838Z	DEBUG	controller-runtime.manager.events	Normal	{"object": {"kind":"ConfigMap","namespace":"opdemo-system","name":"04d31993.ydzs.io","uid":"ddd72f92-8f5c-47d9-84cc-dd40229b0289","apiVersion":"v1","resourceVersion":"4059032"}, "reason": "LeaderElection", "message": "opdemo-controller-manager-57bccc448f-v7blh_e891cd38-e36a-4e2a-a35d-9c1c10f3b118 became leader"}
2021-05-02T04:04:12.838Z	INFO	controller	Starting EventSource	{"reconcilerGroup": "app.ydzs.io", "reconcilerKind": "MyApp", "controller": "myapp", "source": "kind source: /, Kind="}
2021-05-02T04:04:12.939Z	INFO	controller	Starting EventSource	{"reconcilerGroup": "app.ydzs.io", "reconcilerKind": "MyApp", "controller": "myapp", "source": "kind source: /, Kind="}

```
由于我的集群版本过高，还有些小问题，建议使用1.18 1.19 版本的集群 

阳明老师课程地址  
https://youdianzhishi.com/web/course/1018?utm_source=blog&utm_campaign=referral&utm_medium=tophero
