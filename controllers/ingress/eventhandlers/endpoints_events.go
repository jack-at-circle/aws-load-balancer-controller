package eventhandlers

import (
	"context"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/aws-load-balancer-controller/pkg/k8s"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
)

// NewEnqueueRequestsForEndpointsEvent constructs new enqueueRequestsForEndpointsEvent.
func NewEnqueueRequestsForEndpointsEvent(svcEventChan chan<- event.TypedGenericEvent[*corev1.Service], k8sClient client.Client, logger logr.Logger) handler.TypedEventHandler[*corev1.Endpoints] {
	return &enqueueRequestsForEndpointsEvent{
		svcEventChan: svcEventChan,
		k8sClient:    k8sClient,
		logger:       logger,
	}
}

var _ handler.TypedEventHandler[*corev1.Endpoints] = (*enqueueRequestsForEndpointsEvent)(nil)

type enqueueRequestsForEndpointsEvent struct {
	svcEventChan chan<- event.TypedGenericEvent[*corev1.Service]
	k8sClient    client.Client
	logger       logr.Logger
}

// Create is called in response to an create event - e.g. Pod Creation.
func (h *enqueueRequestsForEndpointsEvent) Create(ctx context.Context, e event.TypedCreateEvent[*corev1.Endpoints], queue workqueue.RateLimitingInterface) {
	epNew := e.Object
	h.enqueueImpactedServices(ctx, epNew)
}

// Update is called in response to an update event -  e.g. Pod Updated.
func (h *enqueueRequestsForEndpointsEvent) Update(ctx context.Context, e event.TypedUpdateEvent[*corev1.Endpoints], queue workqueue.RateLimitingInterface) {
	epOld := e.ObjectOld
	epNew := e.ObjectNew
	if !equality.Semantic.DeepEqual(epOld.Subsets, epNew.Subsets) {
		h.enqueueImpactedServices(ctx, epNew)
	}
}

// Delete is called in response to a delete event - e.g. Pod Deleted.
func (h *enqueueRequestsForEndpointsEvent) Delete(ctx context.Context, e event.TypedDeleteEvent[*corev1.Endpoints], queue workqueue.RateLimitingInterface) {
	epOld := e.Object
	h.enqueueImpactedServices(ctx, epOld)
}

// Generic is called in response to an event of an unknown type or a synthetic event triggered as a cron or
// external trigger request - e.g. reconcile AutoScaling, or a WebHook.
func (h *enqueueRequestsForEndpointsEvent) Generic(context.Context, event.TypedGenericEvent[*corev1.Endpoints], workqueue.RateLimitingInterface) {
}

func (h *enqueueRequestsForEndpointsEvent) enqueueImpactedServices(ctx context.Context, ep *corev1.Endpoints) {
	svc := &corev1.Service{}
	if err := h.k8sClient.Get(ctx, client.ObjectKey{Namespace: ep.Namespace, Name: ep.Name}, svc); err != nil {
		h.logger.Error(err, "failed to fetch service")
		return
	}

	epKey := k8s.NamespacedName(ep)
	h.logger.V(1).Info("enqueue service for endpoints event",
		"endpoints", epKey,
		"service", k8s.NamespacedName(&svc.ObjectMeta),
	)
	h.svcEventChan <- event.TypedGenericEvent[*corev1.Service]{
		Object: svc,
	}
}
