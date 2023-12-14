@dedicated-cluster
Feature: Resource creation in Dedicated Cluster

    Scenario: Cluster-scoped Resource can be created
        When Resource is created:
        """
            apiVersion: v1
            kind: Namespace
            metadata:
                name: here-i-can
        """
        Then Resource exists:
        """
            apiVersion: v1
            kind: Namespace
            metadata:
                name: here-i-can
        """

    Scenario: Namespaced-scoped can be created
        Given Resource is created:
        """
            apiVersion: v1
            kind: Namespace
            metadata:
                name: here-i-can
        """
        When Resource is created:
        """
            apiVersion: v1
            kind: Secret
            metadata:
                name: host-operator-controller
                namespace: here-i-can
        """
        Then Resource exists:
        """
            apiVersion: v1
            kind: Secret
            metadata:
                name: host-operator-controller
                namespace: here-i-can
        """
