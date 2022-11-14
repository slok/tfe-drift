<p align="center">
    <img src="docs/img/logo.png" width="20%" align="center" alt="tfe-drift">
</p>

# tfe-drift

[![CI](https://github.com/slok/tfe-drift/actions/workflows/ci.yaml/badge.svg?branch=main)](https://github.com/slok/tfe-drift/actions/workflows/ci.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/slok/tfe-drift)](https://goreportcard.com/report/github.com/slok/tfe-drift)
[![Apache 2 licensed](https://img.shields.io/badge/license-Apache2-blue.svg)](https://raw.githubusercontent.com/slok/tfe-drift/master/LICENSE)
[![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/slok/tfe-drift)](https://github.com/slok/tfe-drift/releases/latest)

## Introduction

Automated Terraform Cloud/Enterprise drift detection.

## Features

- Automate the execution of drift detection plans.
- Limit executed drift detection plans (used to avoid long plan queues with available workers).
- Sort drift detection plans by previous detections age.
- Filter drift detections by workspace.
- Ignore if drift detection plan not required (already running, executed recently...)
- Result of the detection plans summary as output to automate with other apps.
- Easy to automate with CI (e.g Github actions).
- Compatible with Terraform Cloud and Terraform Enterprise.
- Easy and simple to use.

## Getting started

Execute with safe defaults and get the result output in JSON:

```bash
tfe-drift run -o json
```

Limit to a max of 2 executed plans, ignore workspace drift detections that have been already executed in the last 2h, and exclude dns workspace:

```bash
tfe-drift run --exclude dns --not-before 2h --limit-max-plan 2
```

## Exit codes in single run mode

- `0`: If everything is as it should.
- `1`: If there was an error executing tfe-drift.
- `2`: If there was any drift.
- `3`: If there was any error on a drift detection plan.
