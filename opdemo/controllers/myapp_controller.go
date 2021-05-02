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
	if err := r.Client.Get(ctx, req.NamespacedName, &myapp); err != nil {
		// MyApp was deleted, Ignore
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// dedaoMyApp 过后去创建对应的Deployment 和 Service(观察当前状态和期望状态进行对比)
	// 创建就得去判断是否存在，存在就忽略，不存在就创建

	//调谐，获取到当前的一个状态，然后和我们期望的状态进行比对
	// CreateOrUpdate Deployment
	var deploy appsv1.Deployment
	deploy.Name = myapp.Name
	deploy.Namespace = myapp.Namespace
	or, err := ctrl.CreateOrUpdate(ctx, r, &deploy, func() error {
		//调谐必须在函数中实现
		Muatedeployment(&myapp, &deploy)
		return controllerutil.SetControllerReference(&myapp, &deploy, r.Scheme)
	})
	if err != nil {
		return ctrl.Result{}, err
	}
	log.Info("CreateOrUpdate", "Deployment", or)

	//CreateOrUpdate Service
	var service corev1.Service
	service.Name = myapp.Name
	service.Namespace = myapp.Namespace
	or, err = ctrl.CreateOrUpdate(ctx, r, &service, func() error {
		//调谐必须在函数中实现
		MuateService(&myapp, &service)
		return controllerutil.SetControllerReference(&myapp, &service, r.Scheme)
	})
	if err != nil {
		return ctrl.Result{}, err
	}
	log.Info("CreateOrUpdate", "Service", or)
	return ctrl.Result{}, nil
}

func (r *MyAppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		For(&appv1beta1.MyApp{}).
		Complete(r)
}
