package connect

import (
	"anttrader/internal/service"
)

type AdminService struct {
	adminSvc *service.AdminService
}

func NewAdminService(adminSvc *service.AdminService) *AdminService {
	return &AdminService{
		adminSvc: adminSvc,
	}
}
