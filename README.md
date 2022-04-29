# r53tool

A swiss army knife for Route53.

This projects started out as a fork from [route53copy](https://github.com/andersjanmyr/route53copy), but as I added more tools to it, keeping the fork didn't make much sense so new repository and new name for the tool: **r53tool**.

## Installation

Download the lastest version from [releases](https://github.com/pedrokiefer/r53tool/releases) page.

Or build it yourself by cloning this repository.

## Usage

```
$ ./r53tool
r53tool is a swiss army knife for Route53

Usage:
  r53tool [command]

Available Commands:
  check-zone  Check if a zone exists
  completion  Generate the autocompletion script for the specified shell
  copy        Copy is a tool to copy records from one AWS account to another
  delete      Delete is a tool to safely remove a zone and records from Route53
  domains     Domains is a tool to move registered domains from one AWS account to another
  help        Help about any command
  park        Park is a tool to park domains in Route53 creating A and www CNAME records
  version     Print the version number of r53tool

Flags:
      --dry       Dry run
  -h, --help      help for r53tool
  -v, --version   version for r53tool

Use "r53tool [command] --help" for more information about a command.
```
