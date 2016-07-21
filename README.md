# ec2-filter

A small command-line utility to discover EC2 instances and output information
like their private IP address, DNS name, etc.

Inspired by `aws ec2 describe-instances --filters` but easier to deploy and use
for service discovery in scripts.

## Usage

```bash
# Export AWS credentials and region
export AWS_ACCESS_KEY_ID=...
export AWS_SECRET_ACCESS_KEY=...
export AWS_REGION=...

# Show private IPs of all running instances
ec2-filter

# Show private DNS names of all running instances
ec2-filter -private-dns

# Show private IPs of all running instances where tag "foo" is set to "bar"
ec2-filter tag:foo=bar

# Concatenate results using space instead of newline
ec2-filter -join " "

# Limit the number of results, e.g. to only get a single IP
ec2-filter -limit 1

# Use custom format string to print addtional information, e.g. service port
ec2-filter -format '{{.PrivateIpAddress}}:80'

# Combine the presented features
ec2-filter -format '{{.PrivateIpAddress}}:80' -join " " tag:foo=bar
```

Invoke `ec2-filter -h` for a list of all available options.

## Examples from our codebase

Run a command on all EC2 instances that are part of our Crims cluster:

```
pssh -H "$(ec2-filter -join " " tag:jimdo:cluster=crims)" ...
```

Find the endpoint of one Nomad client:

```
NOMAD_ADDR=$(ec2-filter -limit 1 -format 'http://{{.PrivateIpAddress}}:8080' tag:jimdo:cluster=nomad-client)
```
