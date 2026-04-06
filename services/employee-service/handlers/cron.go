package handlers

import (
	"context"
	"log"
	"time"
)

// StartCronJobs launches background cron goroutines for the employee-service.
func (s *EmployeeServer) StartCronJobs() {
	go s.runDailyReset()
}

// runDailyReset resets used_limit = 0 for all agents every day at 23:59 (issue #146).
func (s *EmployeeServer) runDailyReset() {
	for {
		now := time.Now()
		next := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 0, 0, now.Location())
		if !now.Before(next) {
			next = next.Add(24 * time.Hour)
		}
		time.Sleep(time.Until(next))

		log.Println("employee-service: resetting agent used_limit")
		ctx := context.Background()
		res, err := s.DB.ExecContext(ctx, `UPDATE actuary_info SET used_limit = 0`)
		if err != nil {
			log.Printf("employee-service: cron reset error: %v", err)
			continue
		}
		n, _ := res.RowsAffected()
		log.Printf("employee-service: reset used_limit for %d agents", n)
	}
}
