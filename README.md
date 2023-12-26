# Zncdata Stack Operator for Trino

[![Build Status](https://travis-ci.org/zncdata/trino-operator.svg?branch=main)](https://travis-ci.org/zncdata/trino-operator)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](http://www.apache.org/licenses/LICENSE-2.0)
[![codecov](https://codecov.io/gh/zncdata/trino-operator/branch/main/graph/badge.svg)](https://codecov.io/gh/zncdata/trino-operator)

This is a Kubernetes operator to manage [Trino](https://trino.io/) ensembles.

It is part of the Stack ZncData Platform, a curated selection of the best open source data apps like Apache Hive, Apache Druid, Trino or Apache Spark, working together seamlessly. Based on Kubernetes, it runs everywhere.

## Quick Start

1. Install Operator Lifecycle Manager (OLM), a tool to help manage the Operators running on your cluster.

    ```bash
    curl -sL https://github.com/operator-framework/operator-lifecycle-manager/releases/download/v0.26.0/install.sh | bash -s v0.26.0
    ```

2. First we need to prepare an OperatorGroup

    ```bash
    apiVersion: operators.coreos.com/v1
    kind: OperatorGroup
    metadata:
      name: operatorgroup
    spec:
      targetNamespaces:
      - tmp
      upgradeStrategy: Default
    ```

3. Start deploying our catalog

    ```bash
    apiVersion: operators.coreos.com/v1alpha1
    kind: CatalogSource
    metadata:
      name: catalog-v0-0-1-alpha
      namespace: tmp
    spec:
      displayName: zncdata operators
      grpcPodConfig:
        securityContextConfig: restricted
      image: quay.io/zncdata/catalog:v0.0.1-alpha
      publisher: zncdata.net
      sourceType: grpc
      updateStrategy:
        registryPoll:
          interval: 60m
    ```

4. After completing the OperatorGroup and Catalog, you can start installing the service Subscription

    ```bash
    apiVersion: operators.coreos.com/v1alpha1
    kind: Subscription
    metadata:
      name: trino-operator-v0-0-1-alpha-sub
      namespace: tmp
    spec:
      channel: fast-v0.0
      name: trino-operator
      source: catalog
      sourceNamespace: tmp
      installPlanApproval: Automatic
      startingCSV: trino-operator.v0.0.1-alpha
    ```

5. After install, watch your operator come up using next command.

    ```bash
    kubectl get csv -n tmp
    ```

6. Install Instances of Custom Resources:

    ```sh
    kubectl apply -f config/samples/
    ```