package jobs

import (
	"github.com/endeavored/coursewatch/internal/pkg/models"
)

func Start(a *models.App) {
	sdc := courseDetailJob(a)
	go sdc.Run()
	go courseAvailabilityJob(a, sdc).Run()
}
