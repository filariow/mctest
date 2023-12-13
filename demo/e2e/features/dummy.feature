Feature: Dummy

    Scenario: Resource is created
        When Resource is created:
        """
            apiVersion: v1
            kind: Secret
            metadata:
                name: host-operator-controller
                namespace: will-be-overwritten
        """
        Then Resource exists:
        """
            apiVersion: v1
            kind: Secret
            metadata:
                name: host-operator-controller
                namespace: will-be-overwritten
        """

    @dedicated-cluster
    Scenario: Resource is created
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
