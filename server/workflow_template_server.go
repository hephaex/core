package server

import (
	"context"
	"errors"
	"github.com/onepanelio/core/api"
	v1 "github.com/onepanelio/core/pkg"
	"github.com/onepanelio/core/pkg/util/pagination"
	"github.com/onepanelio/core/server/auth"
	"github.com/onepanelio/core/server/converter"
	"strings"
	"time"
)

type WorkflowTemplateServer struct{}

func NewWorkflowTemplateServer() *WorkflowTemplateServer {
	return &WorkflowTemplateServer{}
}

func apiWorkflowTemplate(wft *v1.WorkflowTemplate) *api.WorkflowTemplate {
	res := &api.WorkflowTemplate{
		Uid:        wft.UID,
		CreatedAt:  wft.CreatedAt.UTC().Format(time.RFC3339),
		Name:       wft.Name,
		Version:    wft.Version,
		Versions:   wft.Versions,
		Manifest:   wft.Manifest,
		IsLatest:   wft.IsLatest,
		IsArchived: wft.IsArchived,
		Labels:     converter.MappingToKeyValue(wft.Labels),
	}

	if wft.WorkflowExecutionStatisticReport != nil {
		res.Stats = &api.WorkflowExecutionStatisticReport{
			Total:        wft.WorkflowExecutionStatisticReport.Total,
			LastExecuted: wft.WorkflowExecutionStatisticReport.LastExecuted.Format(time.RFC3339),
			Running:      wft.WorkflowExecutionStatisticReport.Running,
			Completed:    wft.WorkflowExecutionStatisticReport.Completed,
			Failed:       wft.WorkflowExecutionStatisticReport.Failed,
			Terminated:   wft.WorkflowExecutionStatisticReport.Terminated,
		}
	}

	if wft.CronWorkflowsStatisticsReport != nil {
		res.CronStats = &api.CronWorkflowStatisticsReport{
			Total: wft.CronWorkflowsStatisticsReport.Total,
		}
	}

	return res
}

func mapToKeyValue(input map[string]string) []*api.KeyValue {
	var result []*api.KeyValue
	for key, value := range input {
		keyValue := &api.KeyValue{
			Key:   key,
			Value: value,
		}

		result = append(result, keyValue)
	}

	return result
}

func (s *WorkflowTemplateServer) CreateWorkflowTemplate(ctx context.Context, req *api.CreateWorkflowTemplateRequest) (*api.WorkflowTemplate, error) {
	client := ctx.Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "create", "argoproj.io", "workflowtemplates", "")
	if err != nil || !allowed {
		return nil, err
	}
	workflowTemplate := &v1.WorkflowTemplate{
		Name:     req.WorkflowTemplate.Name,
		Manifest: req.WorkflowTemplate.Manifest,
		Labels:   converter.APIKeyValueToLabel(req.WorkflowTemplate.Labels),
	}
	workflowTemplate, err = client.CreateWorkflowTemplate(req.Namespace, workflowTemplate)
	if err != nil {
		return nil, err
	}
	req.WorkflowTemplate.Uid = workflowTemplate.UID
	req.WorkflowTemplate.Version = workflowTemplate.Version

	return req.WorkflowTemplate, nil
}

func (s *WorkflowTemplateServer) CreateWorkflowTemplateVersion(ctx context.Context, req *api.CreateWorkflowTemplateRequest) (*api.WorkflowTemplate, error) {
	client := ctx.Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "create", "argoproj.io", "workflowtemplates", req.WorkflowTemplate.Name)
	if err != nil || !allowed {
		return nil, err
	}

	workflowTemplate := &v1.WorkflowTemplate{
		UID:      req.WorkflowTemplate.Uid,
		Name:     req.WorkflowTemplate.Name,
		Manifest: req.WorkflowTemplate.Manifest,
		Labels:   converter.APIKeyValueToLabel(req.WorkflowTemplate.Labels),
	}

	workflowTemplate, err = client.CreateWorkflowTemplateVersion(req.Namespace, workflowTemplate)
	if err != nil {
		return nil, err
	}
	req.WorkflowTemplate.Uid = workflowTemplate.UID
	req.WorkflowTemplate.Name = workflowTemplate.Name
	req.WorkflowTemplate.Version = workflowTemplate.Version

	return req.WorkflowTemplate, nil
}

func (s *WorkflowTemplateServer) UpdateWorkflowTemplateVersion(ctx context.Context, req *api.UpdateWorkflowTemplateVersionRequest) (*api.WorkflowTemplate, error) {
	client := ctx.Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "update", "argoproj.io", "workflowtemplates", req.WorkflowTemplate.Name)
	if err != nil || !allowed {
		return nil, err
	}

	workflowTemplate := &v1.WorkflowTemplate{
		UID:      req.WorkflowTemplate.Uid,
		Name:     req.WorkflowTemplate.Name,
		Manifest: req.WorkflowTemplate.Manifest,
		Version:  req.WorkflowTemplate.Version,
	}

	req.WorkflowTemplate.Uid = workflowTemplate.UID
	req.WorkflowTemplate.Name = workflowTemplate.Name
	req.WorkflowTemplate.Version = workflowTemplate.Version

	return req.WorkflowTemplate, nil
}

func (s *WorkflowTemplateServer) GetWorkflowTemplate(ctx context.Context, req *api.GetWorkflowTemplateRequest) (*api.WorkflowTemplate, error) {
	client := ctx.Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "get", "argoproj.io", "workflowtemplates", "")
	if err != nil || !allowed {
		return nil, err
	}

	workflowTemplate, err := client.GetWorkflowTemplate(req.Namespace, req.Uid, req.Version)
	if err != nil {
		return nil, err
	}

	return apiWorkflowTemplate(workflowTemplate), nil
}

func (s *WorkflowTemplateServer) CloneWorkflowTemplate(ctx context.Context, req *api.CloneWorkflowTemplateRequest) (*api.WorkflowTemplate, error) {
	client := ctx.Value("kubeClient").(*v1.Client)

	allowed, err := auth.IsAuthorized(client, req.Namespace, "create", "argoproj.io", "workflowtemplates", "")
	if err != nil || !allowed {
		return nil, err
	}

	//Verify the template exists
	workflowTemplate, err := client.GetWorkflowTemplate(req.Namespace, req.Uid, req.Version)
	if err != nil {
		return nil, err
	}

	//Verify the cloned template name doesn't exist already
	workflowTemplateByName, err := client.GetWorkflowTemplateByName(req.Namespace, req.Name, req.Version)
	if err != nil {
		if !strings.Contains(err.Error(), "not found") {
			return nil, err
		}
	}
	if workflowTemplateByName != nil {
		return nil, errors.New("Cannot clone, WorkflowTemplate name already taken.")
	}

	workflowTemplateClone := &v1.WorkflowTemplate{
		Name:     req.Name,
		Manifest: workflowTemplate.Manifest,
		IsLatest: true,
	}
	workflowTemplateCloned, err := client.CreateWorkflowTemplate(req.Namespace, workflowTemplateClone)
	if err != nil {
		return nil, err
	}

	return apiWorkflowTemplate(workflowTemplateCloned), nil
}

func (s *WorkflowTemplateServer) ListWorkflowTemplateVersions(ctx context.Context, req *api.ListWorkflowTemplateVersionsRequest) (*api.ListWorkflowTemplateVersionsResponse, error) {
	client := ctx.Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "list", "argoproj.io", "workflowtemplates", "")
	if err != nil || !allowed {
		return nil, err
	}

	workflowTemplateVersions, err := client.ListWorkflowTemplateVersions(req.Namespace, req.Uid)
	if err != nil {
		return nil, err
	}

	var workflowTemplates []*api.WorkflowTemplate
	for _, wtv := range workflowTemplateVersions {
		workflowTemplates = append(workflowTemplates, apiWorkflowTemplate(wtv))
	}

	return &api.ListWorkflowTemplateVersionsResponse{
		Count:             int32(len(workflowTemplateVersions)),
		WorkflowTemplates: workflowTemplates,
	}, nil
}

func (s *WorkflowTemplateServer) ListWorkflowTemplates(ctx context.Context, req *api.ListWorkflowTemplatesRequest) (*api.ListWorkflowTemplatesResponse, error) {
	client := ctx.Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "list", "argoproj.io", "workflowtemplates", "")
	if err != nil || !allowed {
		return nil, err
	}

	paginator := pagination.NewRequest(req.Page, req.PageSize)
	workflowTemplates, err := client.ListWorkflowTemplates(req.Namespace, &paginator)
	if err != nil {
		return nil, err
	}

	apiWorkflowTemplates := []*api.WorkflowTemplate{}
	for _, wtv := range workflowTemplates {
		apiWorkflowTemplates = append(apiWorkflowTemplates, apiWorkflowTemplate(wtv))
	}

	count, err := client.CountWorkflowTemplates(req.Namespace)
	if err != nil {
		return nil, err
	}

	return &api.ListWorkflowTemplatesResponse{
		Count:             int32(len(apiWorkflowTemplates)),
		WorkflowTemplates: apiWorkflowTemplates,
		Page:              int32(paginator.Page),
		Pages:             paginator.CalculatePages(count),
		TotalCount:        int32(count),
	}, nil
}

func (s *WorkflowTemplateServer) ArchiveWorkflowTemplate(ctx context.Context, req *api.ArchiveWorkflowTemplateRequest) (*api.ArchiveWorkflowTemplateResponse, error) {
	client := ctx.Value("kubeClient").(*v1.Client)
	allowed, err := auth.IsAuthorized(client, req.Namespace, "delete", "argoproj.io", "workflowtemplates", "")
	if err != nil || !allowed {
		return nil, err
	}

	archived, err := client.ArchiveWorkflowTemplate(req.Namespace, req.Uid)
	if err != nil {
		return nil, err
	}

	return &api.ArchiveWorkflowTemplateResponse{
		WorkflowTemplate: &api.WorkflowTemplate{
			IsArchived: archived,
		},
	}, nil
}