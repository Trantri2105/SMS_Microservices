package service

import (
	"VCS_SMS_Microservice/internal/server-service/model"
	"VCS_SMS_Microservice/internal/server-service/repository"
	"VCS_SMS_Microservice/pkg/mail"
	"context"
	"fmt"
	"time"
)

type ServerService interface {
	CreateServer(ctx context.Context, server model.Server) (model.Server, error)
	CreateServers(ctx context.Context, server []model.Server) (insertedServers []model.Server, nonInsertedServer []model.Server, err error)
	UpdateServer(ctx context.Context, updatedServerData model.Server) (model.Server, error)
	DeleteServer(ctx context.Context, id string) error
	GetServers(ctx context.Context, serverName string, status string, sortBy string, sortOrder string, limit int, offset int) ([]model.Server, error)
	ReportServersInformation(ctx context.Context, startDate time.Time, endDate time.Time, mail string) error
	GetServerUptimePercentage(ctx context.Context, serverID string, startDate time.Time, endDate time.Time) (float64, error)
}

type serverService struct {
	serverRepository      repository.ServerRepository
	healthCheckRepository repository.HealthCheckRepository
	mailSender            mail.Sender
}

func (s *serverService) GetServerUptimePercentage(ctx context.Context, serverID string, startDate time.Time, endDate time.Time) (float64, error) {
	res, err := s.healthCheckRepository.GetServerUptimePercentage(ctx, serverID, startDate, endDate)
	if err != nil {
		return 0, fmt.Errorf("ServerService.GetServerUptimePercentage %w", err)
	}
	return res, nil
}

func (s *serverService) ReportServersInformation(ctx context.Context, startDate time.Time, endDate time.Time, mail string) error {
	serversInfo, err := s.healthCheckRepository.GetAllServersHealthInformation(ctx, startDate, endDate)
	if err != nil {
		return fmt.Errorf("ServerService.ReportServersInformation: %w", err)
	}
	textBody := generateTextMailBody(serversInfo)
	htmlBody := generateHTMLBody(serversInfo)
	subject := fmt.Sprintf("Servers Status Report From %s To %s", startDate, endDate.Add(-1*time.Second))
	err = s.mailSender.SendMail([]string{mail}, subject, htmlBody, textBody, nil)
	if err != nil {
		return fmt.Errorf("ServerService.ReportServersInformation: %w", err)
	}
	return nil
}

func generateTextMailBody(serversInfo repository.ServersHealthInformation) string {
	return fmt.Sprintf(
		"--- SUMMARY ---\n"+
			"Total Servers: %d\n"+
			"Healthy: %d\n"+
			"Unhealthy: %d\n"+
			"Inactive: %d\n\n"+
			"Average Uptime Across All Servers: %.2f%%",
		serversInfo.TotalServersCnt,
		serversInfo.HealthyServersCnt,
		serversInfo.UnhealthyServersCnt,
		serversInfo.InactiveServersCnt,
		serversInfo.AverageUptimePercentage,
	)
}

func generateHTMLBody(serversInfo repository.ServersHealthInformation) string {
	htmlFormat := `
<body>
    <table style="width:100%%; border-collapse: collapse;">
        <tr>
            <td style="border: 1px solid #dddddd; text-align: left; padding: 8px; background-color: #f2f2f2;">Total Servers:</td>
            <td style="border: 1px solid #dddddd; text-align: left; padding: 8px;">%d</td>
        </tr>
        <tr>
            <td style="border: 1px solid #dddddd; text-align: left; padding: 8px; background-color: #f2f2f2;">Healthy Servers:</td>
            <td style="border: 1px solid #dddddd; text-align: left; padding: 8px;">%d</td>
        </tr>
        <tr>
            <td style="border: 1px solid #dddddd; text-align: left; padding: 8px; background-color: #f2f2f2;">Unhealthy Servers:</td>
            <td style="border: 1px solid #dddddd; text-align: left; padding: 8px;">%d</td>
        </tr>
        <tr>
            <td style="border: 1px solid #dddddd; text-align: left; padding: 8px; background-color: #f2f2f2;">Inactive Servers:</td>
            <td style="border: 1px solid #dddddd; text-align: left; padding: 8px;">%d</td>
        </tr>
        <tr>
            <td style="border: 1px solid #dddddd; text-align: left; padding: 8px; background-color: #f2f2f2;">Average Uptime Percentage:</td>
            <td style="border: 1px solid #dddddd; text-align: left; padding: 8px;">%.2f%%</td>
        </tr>
    </table>
</body>`

	return fmt.Sprintf(htmlFormat,
		serversInfo.TotalServersCnt,
		serversInfo.HealthyServersCnt,
		serversInfo.UnhealthyServersCnt,
		serversInfo.InactiveServersCnt,
		serversInfo.AverageUptimePercentage,
	)
}

func (s *serverService) GetServers(ctx context.Context, serverName string, status string, sortBy string, sortOrder string, limit int, offset int) ([]model.Server, error) {
	servers, err := s.serverRepository.GetServers(ctx, serverName, status, sortBy, sortOrder, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("ServerService.GetServers: %w", err)
	}
	return servers, nil
}

func (s *serverService) CreateServer(ctx context.Context, server model.Server) (model.Server, error) {
	server.Status = model.ServerStatusPending
	createdServer, err := s.serverRepository.CreateServer(ctx, server)
	if err != nil {
		return server, fmt.Errorf("ServerService.CreateServer: %w", err)
	}
	return createdServer, nil
}

func (s *serverService) CreateServers(ctx context.Context, servers []model.Server) (insertedServers []model.Server, nonInsertedServers []model.Server, err error) {
	for i := range servers {
		servers[i].Status = model.ServerStatusPending
	}
	insertedServers, nonInsertedServers, err = s.serverRepository.ImportServers(ctx, servers)
	if err != nil {
		err = fmt.Errorf("ServerService.CreateServers: %w", err)
	}
	return
}

func (s *serverService) UpdateServer(ctx context.Context, updatedServerData model.Server) (model.Server, error) {
	updatedServer, err := s.serverRepository.UpdateServer(ctx, updatedServerData)
	if err != nil {
		return model.Server{}, fmt.Errorf("ServerService.UpdateServer: %w", err)
	}
	return updatedServer, nil
}

func (s *serverService) DeleteServer(ctx context.Context, id string) error {
	err := s.serverRepository.DeleteServerById(ctx, id)
	if err != nil {
		return fmt.Errorf("ServerService.DeleteServer: %w", err)
	}
	return nil
}

func NewServerService(serverRepository repository.ServerRepository, healthCheckRepository repository.HealthCheckRepository, mailSender mail.Sender) ServerService {
	return &serverService{
		serverRepository:      serverRepository,
		healthCheckRepository: healthCheckRepository,
		mailSender:            mailSender,
	}
}
