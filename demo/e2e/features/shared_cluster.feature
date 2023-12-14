Feature: Resource creation in Cluster shared among multiple scenario

    Scenario: Namespaced Resource are successfully created in scenario namespace
        When Resource is created:
        """
            apiVersion: v1
            kind: Secret
            metadata:
                name: host-operator-controller
                namespace: will-be-overwritten
        """
        Then Resource exists in scenario namespace:
        """
            apiVersion: v1
            kind: Secret
            metadata:
                name: host-operator-controller
        """

    Scenario: Cluster-scoped resource can not be created
        Then Resource can not be created:
        """
        apiVersion: v1
        kind: Namespace
        metadata:
            name: notpermitted
        """
