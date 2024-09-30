/*
Copyright 2024.

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
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	k8sopwebgamev1 "github.com/bronzels/k8sopwebgame/api/v1"
)

// WebGameReconciler reconciles a WebGame object
type WebGameReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=k8sopwebgame.op.k8s.at.bronzels,resources=webgames,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=k8sopwebgame.op.k8s.at.bronzels,resources=webgames/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=k8sopwebgame.op.k8s.at.bronzels,resources=webgames/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the WebGame object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *WebGameReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// _ = log.FromContext(ctx)

	// TODO(user): your logic here
	logger := log.FromContext(ctx)
	logger.V(2).Info("webgame event received")
	defer func() { logger.V(2).Info("webgame event handling completed") }()

	var webgame k8sopwebgamev1.WebGame
	if err := r.Get(ctx, req.NamespacedName, &webgame); err != nil {
		// 获取webgame失败，并且错误原因是Not Found
		// 说明触发本次Reconcile的是webgame的删除时间，不需要进一步处理，return即可
		if errors.IsNotFound(err) {
			logger.Info("webgame not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// 获取webgame失败，但是错误原因不是Not Found
		// 返回这个错误，触发本次Reconcile的时间会重新入队，等待重试
		// 同时日志会打印本次的错误
		logger.Error(err, "unable to find webgame, requeue")
		return ctrl.Result{}, err
	}

	// 成功获取到webgame实例，说明触发本次Reconcile的是webgame的创建或更新事件
	// 在日志中打印创建/更新的 webgame 资源的name

	selector := map[string]string{
		"gameType": webgame.Spec.GameType,
		"instance": webgame.GetName(),
	}

	var deployment = appsv1.Deployment{}
	deployment.SetNamespace(webgame.GetNamespace())
	deployment.SetName(webgame.GetName())

	mutate := func() error {
		deployment.SetLabels(labels.Merge(deployment.GetLabels(), webgame.GetLabels()))
		deployment.Spec.Replicas = webgame.Spec.Replicas
		deployment.Spec.Selector = &metav1.LabelSelector{MatchLabels: selector}
		deployment.Spec.Template.SetLabels(labels.Merge(webgame.GetLabels(), selector))

		container := corev1.Container{}
		if len(deployment.Spec.Template.Spec.Containers) != 0 {
			container = deployment.Spec.Template.Spec.Containers[0]
		}
		container.Name = webgame.GetName()
		container.Image = webgame.Spec.Image
		container.ImagePullPolicy = corev1.PullIfNotPresent
		container.Resources = corev1.ResourceRequirements{}
		container.Ports = []corev1.ContainerPort{{
			Name:          "web",
			ContainerPort: int32(webgame.Spec.ServerPort.IntValue()),
			Protocol:      corev1.ProtocolTCP,
		}}

		deployment.Spec.Template.Spec.Containers = []corev1.Container{container}
		return ctrl.SetControllerReference(&webgame, &deployment, r.Scheme)
	}

	res, err := ctrl.CreateOrUpdate(ctx, r.Client, &deployment, mutate)
	if err != nil {
		return ctrl.Result{}, err
	}

	if res != controllerutil.OperationResultNone {
		logger.Info("deployment changed", "res", res)
		return ctrl.Result{}, nil
	}

	// create service
	var service = corev1.Service{}
	service.SetNamespace(webgame.GetNamespace())
	service.SetName(webgame.GetName())
	mutate = func() error {
		service.SetLabels(labels.Merge(service.GetLabels(), webgame.GetLabels()))
		service.Spec.Selector = selector
		service.Spec.Type = corev1.ServiceTypeClusterIP
		service.Spec.Ports = []corev1.ServicePort{{
			Name:       "web",
			Port:       int32(webgame.Spec.ServerPort.IntValue()),
			TargetPort: webgame.Spec.ServerPort,
			Protocol:   corev1.ProtocolTCP,
		}}

		return ctrl.SetControllerReference(&webgame, &service, r.Scheme)
	}

	res, err = ctrl.CreateOrUpdate(ctx, r.Client, &service, mutate)
	if err != nil {
		return ctrl.Result{}, err
	}

	if res != controllerutil.OperationResultNone {
		logger.Info("service changed", "res", res)
		return ctrl.Result{}, nil
	}

	// create ingress
	var (
		ingress     = networkingv1.Ingress{}
		pathType    = networkingv1.PathTypePrefix
		path        = fmt.Sprintf("/%s/%s", selector["gameType"], selector["instance"])
		rewriteRule = fmt.Sprintf(`rewrite ^%s/(.*)$ /$ break;`, path)
		annotations = map[string]string{
			"nginx.ingress.kubernetes.io/configuration-snippet": rewriteRule,
		}
	)
	ingress.SetNamespace(webgame.GetNamespace())
	ingress.SetName(webgame.GetName())

	mutate = func() error {
		ingress.SetLabels(labels.Merge(ingress.GetLabels(), webgame.GetLabels()))
		ingress.SetAnnotations(labels.Merge(ingress.GetAnnotations(), annotations))
		ingress.Spec = networkingv1.IngressSpec{
			IngressClassName: &webgame.Spec.IngressClass,
			Rules: []networkingv1.IngressRule{{
				IngressRuleValue: networkingv1.IngressRuleValue{
					HTTP: &networkingv1.HTTPIngressRuleValue{
						Paths: []networkingv1.HTTPIngressPath{{
							PathType: &pathType,
							Path:     path,
							Backend: networkingv1.IngressBackend{
								Service: &networkingv1.IngressServiceBackend{
									Name: service.GetName(),
									Port: networkingv1.ServiceBackendPort{
										Number: int32(webgame.Spec.ServerPort.IntValue()),
									},
								},
							},
						}},
					},
				},
			}},
		}

		return controllerutil.SetControllerReference(&webgame, &ingress, r.Scheme)
	}

	res, err = ctrl.CreateOrUpdate(ctx, r.Client, &ingress, mutate)
	if err != nil {
		return ctrl.Result{}, err
	}

	if res != controllerutil.OperationResultNone {
		logger.Info("ingress changed", "res", res)
		return ctrl.Result{}, nil
	}

	logger.Info("sync status")
	mutate = func() error {
		index := strings.TrimPrefix(webgame.Spec.IndexPage, "/")
		path := strings.TrimPrefix(path, "/")
		address := fmt.Sprintf("%s/%s/%s", webgame.Spec.Domain, path, index)
		webgame.Status.DeploymentStatus = *deployment.Status.DeepCopy()
		webgame.Status.GameAddress = address
		webgame.Status.ClusterIP = service.Spec.ClusterIP
		return nil
	}

	res, err = controllerutil.CreateOrPatch(ctx, r.Client, &webgame, mutate)
	if err != nil {
		return ctrl.Result{}, err
	}

	if res != controllerutil.OperationResultNone {
		logger.Info("webgame status synced", "res", res)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *WebGameReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sopwebgamev1.WebGame{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&networkingv1.Ingress{}).
		Complete(r)
}
