package management

import (
	"context"
	"errors"
	"fmt"
)

type Action string

const (
	ActionDeploy   Action = "deploy"
	ActionRedeploy Action = "redeploy"
	ActionRestart  Action = "restart"
	ActionStop     Action = "stop"
	ActionDelete   Action = "delete"
)

var ErrForbidden = errors.New("bu işlem için yetkiniz yok")

type Actor struct {
	ID      string
	Role    string
	Channel string
	IP      string
}

type Request struct {
	Actor      Actor
	Action     Action
	ResourceID string
	Resource   string
}

type Result struct {
	Message      string
	OperationID  string
	AuditPending bool
}

type Operations struct {
	Deploy    func(context.Context, string, bool) (string, error)
	Restart   func(context.Context, string) error
	Stop      func(context.Context, string) error
	Delete    func(context.Context, string) error
	Authorize func(role string, action Action) bool
}

type Service struct {
	operations Operations
}

func NewService(operations Operations) *Service {
	return &Service{operations: operations}
}

func (service *Service) Execute(ctx context.Context, request Request) (Result, error) {
	if request.Actor.ID == "" || request.ResourceID == "" {
		return Result{}, errors.New("kullanıcı ve kaynak bilgisi zorunludur")
	}
	if service.operations.Authorize == nil || !service.operations.Authorize(request.Actor.Role, request.Action) {
		return Result{}, ErrForbidden
	}

	switch request.Action {
	case ActionDeploy, ActionRedeploy:
		if service.operations.Deploy == nil {
			return Result{}, errors.New("deploy işlemi yapılandırılmadı")
		}
		operationID, err := service.operations.Deploy(ctx, request.ResourceID, request.Action == ActionRedeploy)
		return Result{Message: "deployment kuyruğa alındı", OperationID: operationID, AuditPending: true}, err
	case ActionRestart:
		return Result{Message: "yeniden başlatma kuyruğa alındı", AuditPending: true}, service.call(service.operations.Restart, ctx, request.ResourceID, "restart")
	case ActionStop:
		return Result{Message: "durdurma kuyruğa alındı", AuditPending: true}, service.call(service.operations.Stop, ctx, request.ResourceID, "stop")
	case ActionDelete:
		return Result{Message: "kaynak silindi", AuditPending: true}, service.call(service.operations.Delete, ctx, request.ResourceID, "delete")
	default:
		return Result{}, fmt.Errorf("desteklenmeyen işlem: %s", request.Action)
	}
}

func (service *Service) call(operation func(context.Context, string) error, ctx context.Context, resourceID, name string) error {
	if operation == nil {
		return fmt.Errorf("%s işlemi yapılandırılmadı", name)
	}
	return operation(ctx, resourceID)
}
