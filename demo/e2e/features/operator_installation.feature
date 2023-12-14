@operator
Feature: Operator installation

    Scenario: Operator is installed in shared cluster
        When Operator "show" is installed
        Then Resource exists:
        """
        apiVersion: apps/v1
        kind: Deployment
        metadata:
            name: controller-manager
        """

    @dedicated-cluster
    Scenario: Operator is installed in shared cluster
        Given Resource is created:
        """
            apiVersion: v1
            kind: Namespace
            metadata:
                name: system
        """
        When Operator "show" is installed in namespace "system"
        Then Resource exists:
        """
        apiVersion: apps/v1
        kind: Deployment
        metadata:
            name: controller-manager
            namespace: system
        """
