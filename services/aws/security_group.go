package aws

import (
	"context"

	awsConfig "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	. "github.com/mlabouardy/komiser/models/aws"
)

func (aws AWS) DescribeSecurityGroupsTotal(cfg awsConfig.Config) (int64, error) {
	var sum int64
	regions, err := aws.getRegions(cfg)
	if err != nil {
		return 0, err
	}
	for _, region := range regions {
		sgs, err := aws.getSecurityGroups(cfg, region.Name)
		if err != nil {
			return 0, err
		}
		sum += int64(len(sgs))
	}
	return sum, nil
}

func (aws AWS) getSecurityGroups(cfg awsConfig.Config, region string) ([]SecurityGroup, error) {
	cfg.Region = region
	svc := ec2.NewFromConfig(cfg)
	result, err := svc.DescribeSecurityGroups(context.Background(), &ec2.DescribeSecurityGroupsInput{})
	if err != nil {
		return []SecurityGroup{}, err
	}
	listOfSecurityGroups := make([]SecurityGroup, 0)
	for _, securityGroup := range result.SecurityGroups {
		sgTags := make([]string, 0)
		for _, tag := range securityGroup.Tags {
			sgTags = append(sgTags, *tag.Value)
		}
		listOfSecurityGroups = append(listOfSecurityGroups, SecurityGroup{
			Tags: sgTags,
		})
	}
	return listOfSecurityGroups, nil
}

type UnrestrictedSecurityGroup struct {
	Region   string
	Name     string
	ID       string
	Protocol string
	FromPort int64
	ToPort   int64
}

func (awsClient AWS) ListUnrestrictedSecurityGroups(cfg awsConfig.Config) ([]UnrestrictedSecurityGroup, error) {
	groups := make([]UnrestrictedSecurityGroup, 0)

	regions, err := awsClient.getRegions(cfg)
	if err != nil {
		return groups, err
	}
	for _, region := range regions {
		cfg.Region = region.Name
		svc := ec2.NewFromConfig(cfg)
		res, err := svc.DescribeSecurityGroups(context.Background(), &ec2.DescribeSecurityGroupsInput{})
		if err != nil {
			return groups, err
		}

		for _, sg := range res.SecurityGroups {
			for _, permission := range sg.IpPermissions {
				for _, ip := range permission.IpRanges {
					if *ip.CidrIp == "0.0.0.0/0" {
						if *permission.IpProtocol == "-1" {
							groups = append(groups, UnrestrictedSecurityGroup{
								Region:   region.Name,
								Name:     *sg.GroupName,
								ID:       *sg.GroupId,
								Protocol: *permission.IpProtocol,
							})
						} else {
							groups = append(groups, UnrestrictedSecurityGroup{
								Region:   region.Name,
								Name:     *sg.GroupName,
								ID:       *sg.GroupId,
								Protocol: *permission.IpProtocol,
								FromPort: int64(*permission.FromPort),
								ToPort:   int64(*permission.ToPort),
							})
						}

					}
				}
			}
		}

	}
	return groups, nil
}
