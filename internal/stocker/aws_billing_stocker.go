package stocker

import (
	"context"
	"strconv"
	"time"

	cp "github.com/RHEcosystemAppEng/cluster-iq/internal/cloud_providers/aws"
	"github.com/RHEcosystemAppEng/cluster-iq/internal/inventory"
	"go.uber.org/zap"
)

// AWSBillingStocker object to obtain costs and expenses from AWS Cost Explorer API
type AWSBillingStocker struct {
	// Account to scan on this stocker
	Account *inventory.Account
	// Stocker Logger
	logger *zap.Logger
	// AWS connection interface
	conn *cp.AWSConnection
	// List of instance IDs to obtain their expenses
	InstanceIDs []string
}

// NewAWSBillingStocker create and returns a pointer to a new AWSBillingStocker instance
func NewAWSBillingStocker(account *inventory.Account, logger *zap.Logger, instanceIDs []string) *AWSBillingStocker {
	if len(instanceIDs) == 0 {
		logger.Info("No instances pending billing update, skipping billing stocker")
		return nil
	}

	conn, err := cp.NewAWSConnection(context.Background(), account.User(), account.Password(), "", cp.WithCostExplorer())
	if err != nil {
		logger.Error("Error creating a new AWSBillingStocker", zap.String("account", account.AccountName), zap.Error(err))
		return nil
	}

	return &AWSBillingStocker{
		Account:     account,
		logger:      logger,
		InstanceIDs: instanceIDs,
		conn:        conn,
	}
}

// Connect initialices the AWS API and CostExplorer sessions and clients
func (s *AWSBillingStocker) Connect() error {
	s.logger.Info("AWS Session created", zap.String("account", s.Account.AccountName))
	return nil
}

// MakeStock implements the Stocker interface. It starts the Stocker main
// process getting the expenses of the instances stored in the Stocker object
func (s *AWSBillingStocker) MakeStock() error {
	for i := range s.Account.Clusters {
		cluster := s.Account.Clusters[i]
		for j := range cluster.Instances {
			instance := &cluster.Instances[j]
			for _, targetID := range s.InstanceIDs {
				if targetID == instance.InstanceID {
					s.logger.Info("Getting expenses for instance", zap.String("instance_id", targetID))
					err := s.getInstanceExpenses(instance)
					if err != nil {
						s.logger.Error("Error querying billing info for an instance",
							zap.String("account", s.Account.AccountID),
							zap.String("instance_id", instance.InstanceID),
							zap.String("error", err.Error()),
						)
						continue
					}
					break
				}
			}
		}
	}

	return nil
}

// getInstanceExpenses gets from the AWS CostExplorer API the expenses of a given Instance.
func (s *AWSBillingStocker) getInstanceExpenses(instance *inventory.Instance) error {
	// 14-day rolling window: the maximum range AWS Cost Explorer supports at daily resource-level granularity.
	// UTC is required because AWS Cost Explorer uses UTC internally for date boundaries.
	now := time.Now().UTC()
	startDate := now.AddDate(0, 0, -14).Format("2006-01-02")
	endDate := now.Format("2006-01-02")

	s.logger.Debug("Getting expenses for instance",
		zap.String("account", s.Account.AccountName),
		zap.String("instance_id", instance.InstanceID),
		zap.String("start_date", startDate),
		zap.String("end_date", endDate),
	)

	input := &cp.CostAndUsageInput{
		StartDate:  startDate,
		EndDate:    endDate,
		InstanceID: instance.InstanceID,
	}

	result, err := s.conn.CostExplorer.GetCostAndUsageWithResources(context.Background(), input)
	if err != nil {
		s.logger.Error("Error getting cost and usage with resources",
			zap.String("account", s.Account.AccountName),
			zap.String("instance_id", instance.InstanceID),
			zap.Error(err))
		return err
	}

	for _, costResult := range result.Results {
		amount, err := strconv.ParseFloat(costResult.Amount, 64)
		if err != nil {
			s.logger.Error("Error parsing cost amount",
				zap.String("account", s.Account.AccountName),
				zap.Float64("amount", amount),
				zap.Error(err))
			return err
		}

		// AWS Cost Explorer DateInterval uses pattern (\d{4}-\d{2}-\d{2})(T\d{2}:\d{2}:\d{2}Z)?
		// DAILY granularity typically returns "YYYY-MM-DD" but may include "T00:00:00Z".
		expenseDate, err := time.Parse(time.RFC3339, costResult.StartDate)
		if err != nil {
			expenseDate, err = time.Parse("2006-01-02", costResult.StartDate)
		}
		if err != nil {
			s.logger.Error("Error parsing start date",
				zap.String("account", s.Account.AccountName),
				zap.String("start", costResult.StartDate),
				zap.Error(err))
			return err
		}

		// NewExpense always returns a valid pointer; negative amounts are clamped to 0.0
		// by the constructor. AWS Cost Explorer does not return negative costs for instances.
		expense := inventory.NewExpense(instance.InstanceID, amount, expenseDate)
		if err := instance.AddExpense(expense); err != nil {
			s.logger.Error("error when adding an expense to an instance",
				zap.String("instance_id", instance.InstanceID),
				zap.Error(err),
			)
			continue
		}
	}

	return nil
}

// GetAccount returns the account configured for this stocker
func (s AWSBillingStocker) GetAccount() inventory.Account {
	return *s.Account
}
