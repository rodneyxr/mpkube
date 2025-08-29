# clusters

A CLI tool for managing lightweight k3s Kubernetes clusters using Multipass VMs.

## Features

- Create, list, and delete k3s clusters (Multipass VMs)
- Supports Windows, WSL, and Linux environments
- Automatically detects Multipass installation location

## Requirements

- [Multipass](https://multipass.run/) installed and available in your environment

## Installation

```sh
go install github.com/rodneyxr/mpkube
```

## Usage

List available commands:

```sh
mpkube --help
```

### Create a cluster

```sh
mpkube create <mpkube-name>
```

### List clusters

```sh
mpkube list
```

### Delete a cluster

```sh
mpkube delete <mpkube-name>
```
