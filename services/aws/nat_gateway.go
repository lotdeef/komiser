package aws

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	. "github.com/mlabouardy/komiser/models/aws"
)

func (aws AWS) DescribeNatGatewaysTotal(cfg aws.Config) (int64, error) {
	var sum int64
	regions, err := aws.getRegions(cfg)
	if err != nil {
		return 0, err
	}
	for _, region := range regions {
		ngws, err := aws.getNatGateways(cfg, region.Name)
		if err != nil {
			return 0, err
		}
		sum += int64(len(ngws))
	}
	return sum, nil
}

func (aws AWS) getNatGateways(cfg aws.Config, region string) ([]NatGateway, error) {
	cfg.Region = region
	svc := ec2.NewFromConfig(cfg)
	result, err := svc.DescribeNatGateways(context.Background(), &ec2.DescribeNatGatewaysInput{})
	if err != nil {
		return []NatGateway{}, err
	}
	listOfNatGateways := make([]NatGateway, 0)
	for _, ngw := range result.NatGateways {
		ngwTags := make([]string, 0)
		for _, tag := range ngw.Tags {
			ngwTags = append(ngwTags, *tag.Value)
		}
		listOfNatGateways = append(listOfNatGateways, NatGateway{
			ID:    *ngw.NatGatewayId,
			State: string(ngw.State),
			Tags:  ngwTags,
		})
	}
	return listOfNatGateways, nil
}

type NatGatewayMetric struct {
	BytesOutToDestination  float64
	BytesInFromDestination float64
}

func (awsModel AWS) GetNatGatewayTraffic(cfg aws.Config) (map[string]map[string]NatGatewayMetric, error) {
	metrics := make(map[string]map[string]NatGatewayMetric, 0)

	regions, err := awsModel.getRegions(cfg)
	if err != nil {
		return metrics, err
	}
	for _, region := range regions {
		cfg.Region = region.Name
		svc := ec2.NewFromConfig(cfg)
		result, err := svc.DescribeNatGateways(context.Background(), &ec2.DescribeNatGatewaysInput{})
		if err != nil {
			return metrics, err
		}
		for _, ngw := range result.NatGateways {
			cloudwatchService := cloudwatch.NewFromConfig(cfg)
			result, err := cloudwatchService.GetMetricStatistics(context.Background(), &cloudwatch.GetMetricStatisticsInput{
				Namespace:  aws.String("AWS/NATGateway"),
				MetricName: aws.String("BytesOutToDestination"),
				StartTime:  aws.Time(time.Now().AddDate(0, 0, -7)),
				EndTime:    aws.Time(time.Now()),
				Period:     aws.Int32(86400),
				Statistics: []types.Statistic{
					types.StatisticSum,
				},
				Dimensions: []types.Dimension{
					types.Dimension{
						Name:  aws.String("NatGatewayId"),
						Value: ngw.NatGatewayId,
					},
				},
			})
			if err != nil {
				return metrics, err
			}
			for _, metric := range result.Datapoints {
				if metrics[region.Name] == nil {
					metrics[region.Name] = make(map[string]NatGatewayMetric, 0)
					metrics[region.Name][(*metric.Timestamp).Format("2006-01-02")] = NatGatewayMetric{
						BytesOutToDestination: *metric.Sum,
					}
				} else {
					metrics[region.Name][(*metric.Timestamp).Format("2006-01-02")] = NatGatewayMetric{
						BytesOutToDestination: metrics[region.Name][(*metric.Timestamp).Format("2006-01-02")].BytesOutToDestination + *metric.Sum,
					}
				}
			}

			result2, err := cloudwatchService.GetMetricStatistics(context.Background(), &cloudwatch.GetMetricStatisticsInput{
				Namespace:  aws.String("AWS/NATGateway"),
				MetricName: aws.String("BytesInFromDestination"),
				StartTime:  aws.Time(time.Now().AddDate(0, 0, -7)),
				EndTime:    aws.Time(time.Now()),
				Period:     aws.Int32(86400),
				Statistics: []types.Statistic{
					types.StatisticSum,
				},
				Dimensions: []types.Dimension{
					types.Dimension{
						Name:  aws.String("NatGatewayId"),
						Value: ngw.NatGatewayId,
					},
				},
			})
			if err != nil {
				return metrics, err
			}
			for _, metric := range result2.Datapoints {
				if metrics[region.Name] == nil {
					metrics[region.Name] = make(map[string]NatGatewayMetric, 0)
					metrics[region.Name][(*metric.Timestamp).Format("2006-01-02")] = NatGatewayMetric{
						BytesInFromDestination: *metric.Sum,
					}
				} else {
					metrics[region.Name][(*metric.Timestamp).Format("2006-01-02")] = NatGatewayMetric{
						BytesInFromDestination: metrics[region.Name][(*metric.Timestamp).Format("2006-01-02")].BytesInFromDestination + *metric.Sum,
						BytesOutToDestination:  metrics[region.Name][(*metric.Timestamp).Format("2006-01-02")].BytesOutToDestination,
					}
				}
			}
		}
	}
	return metrics, err
}
