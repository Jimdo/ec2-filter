package main

import (
	"bytes"
	"flag"
	"fmt"
	"html/template"
	"io"
	"net"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type EC2Instance struct {
	InstanceId       string
	PrivateDnsName   string
	PrivateIpAddress string
	PublicDnsName    string
	PublicIpAddress  string
}

func main() {
	var (
		showInstanceID = flag.Bool("instance-id", false, "show EC2 instance ID")
		showPrivateDNS = flag.Bool("private-dns", false, "show private DNS name")
		showPrivateIP  = flag.Bool("private-ip", false, "show private IP address")
		showPublicDNS  = flag.Bool("public-dns", false, "show public DNS name")
		showPublicIP   = flag.Bool("public-ip", false, "show public IP address")
		region         = flag.String("region", "", "set AWS region")

		format = flag.String("format", "{{.PrivateIpAddress}}", "alternate format in Go template syntax")
		join   = flag.String("join", "\n", "separator string for concatenating results")
		limit  = flag.Int("limit", 0, "limit number of results")
		output = flag.String("output", "", "write results to given file")
	)
	flag.Parse()

	switch {
	case *showInstanceID:
		*format = "{{.InstanceId}}"
	case *showPrivateDNS:
		*format = "{{.PrivateDnsName}}"
	case *showPrivateIP:
		*format = "{{.PrivateIpAddress}}"
	case *showPublicDNS:
		*format = "{{.PublicDnsName}}"
	case *showPublicIP:
		*format = "{{.PublicIpAddress}}"
	}

	tmpl, err := template.New("main").Parse(*format)
	if err != nil {
		abort("%s", err)
	}

	filters := map[string]string{
		// Only show running EC2 instances by default
		"instance-state-name": "running",
	}
	for _, arg := range flag.Args() {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) != 2 {
			abort("format of filter must be key=value")
		}
		filters[parts[0]] = parts[1]
	}

	instances, err := findInstances(filters, *region)
	if err != nil {
		abort("%s", err)
	}

	var lines []string
	for _, i := range instances {
		var out bytes.Buffer
		if err := tmpl.Execute(&out, i); err != nil {
			abort("%s", err)
		}
		if s := out.String(); len(s) > 0 {
			lines = append(lines, s)
		}
	}

	var w io.Writer = os.Stdout
	if *output != "" {
		f, err := os.Create(*output)
		if err != nil {
			abort("%s", err)
		}
		defer f.Close()
		w = f
	}

	if len(lines) > 0 {
		sort.Sort(Lines(lines))
		maxLines := len(lines)
		if *limit > 0 && *limit < maxLines {
			maxLines = *limit
		}
		fmt.Fprintln(w, strings.Join(lines[:maxLines], *join))
	}
}

func findInstances(filters map[string]string, region string) ([]EC2Instance, error) {
	var ec2Filters []*ec2.Filter
	for k, v := range filters {
		ec2Filters = append(ec2Filters, &ec2.Filter{
			Name:   aws.String(k),
			Values: aws.StringSlice([]string{v}),
		})
	}

	var instances []EC2Instance
	fn := func(output *ec2.DescribeInstancesOutput, last bool) bool {
		for _, r := range output.Reservations {
			for _, i := range r.Instances {
				instances = append(instances, EC2Instance{
					InstanceId:       aws.StringValue(i.InstanceId),
					PrivateDnsName:   aws.StringValue(i.PrivateDnsName),
					PrivateIpAddress: aws.StringValue(i.PrivateIpAddress),
					PublicDnsName:    aws.StringValue(i.PublicDnsName),
					PublicIpAddress:  aws.StringValue(i.PublicIpAddress),
				})
			}
		}
		return !last
	}

	config := aws.NewConfig().WithHTTPClient(&http.Client{Timeout: 30 * time.Second})
	if region != "" {
		config.WithRegion(region)
	}

	sess, err := session.NewSession(config)
	if err != nil {
		return nil, err
	}

	svc := ec2.New(sess)
	if err := svc.DescribeInstancesPages(&ec2.DescribeInstancesInput{Filters: ec2Filters}, fn); err != nil {
		return nil, err
	}

	return instances, nil
}

func abort(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", a...)
	os.Exit(1)
}

type Lines []string

func (l Lines) Len() int { return len(l) }
func (l Lines) Less(i, j int) bool {
	// Compare octets if lines are IP addresses
	if a, b := net.ParseIP(l[i]), net.ParseIP(l[j]); a != nil && b != nil {
		return bytes.Compare(a, b) < 0
	}
	// Otherwise, use string comparison
	return l[i] < l[j]
}
func (l Lines) Swap(i, j int) { l[i], l[j] = l[j], l[i] }
