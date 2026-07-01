package stocker

import (
	"context"
	"strings"
	"time"

	cpaws "github.com/RHEcosystemAppEng/cluster-iq/internal/cloud_providers/aws"
	"github.com/RHEcosystemAppEng/cluster-iq/internal/inventory"
	"go.uber.org/zap"
)

const (
	// consoleProtocolPrefix defines the "HTTP" protocol header
	consoleProtocolPrefix = "https://"
	// consoleLinkPrefix is the pre-defined hostname for the Openshift Console
	consoleLinkPrefix      = "console-openshift-console.apps."
	unknownConsoleLinkCode = ""
)

// generateConsoleLink attaches the consoleLinkPrefix to the baseDomain specified by args
func generateConsoleLink(baseDomain string) string {
	return consoleProtocolPrefix + consoleLinkPrefix + baseDomain
}

// searchConsoleURLinDNSRecords looks for the console link on the record list
func searchConsoleURLinDNSRecords(records []cpaws.DNSRecordSet, cluster *inventory.Cluster) string {
	for _, record := range records {
		if strings.Contains(record.Name, cluster.ClusterName) {
			return generateConsoleLink(record.Name)
		}
	}
	return unknownConsoleLinkCode
}

// getConsoleLinkOfCluster returns the corresponding ConsoleLink for a given cluster
func (s *AWSStocker) getConsoleLinkOfCluster(cluster *inventory.Cluster, hostedZone cpaws.DNSHostedZone) string {
	records, err := s.conn.Route53.GetHostedZoneRecords(context.Background(), hostedZone.ID)
	if err != nil {
		return unknownConsoleLinkCode
	}

	return searchConsoleURLinDNSRecords(records, cluster)
}

// FindOpenshiftConsoleURLs iterates every Cluster and every Route53 HostedZone for looking for the corresponding URLs for the OCP console
func (s *AWSStocker) FindOpenshiftConsoleURLs() error {
	start := time.Now()
	hostedZones, err := s.conn.Route53.GetZonesWithTags(context.Background())
	if err != nil {
		return err
	}
	for i, cluster := range s.Account.Clusters {
		for _, hostedZone := range hostedZones {
			if s.conn.Route53.ZoneBelongsToCluster(cluster, hostedZone) {
				s.logger.Debug("Found Hosted Zone for Cluster", zap.String("account_id", s.Account.AccountID), zap.String("hosted_zone_id", hostedZone.Name), zap.String("cluster_id", cluster.ClusterID))

				s.Account.Clusters[i].ConsoleLink = s.getConsoleLinkOfCluster(cluster, hostedZone)
			}
		}
	}
	s.logger.Debug("Finished finding OpenShift console URLs",
		zap.Duration("duration", time.Since(start)))
	return nil
}
