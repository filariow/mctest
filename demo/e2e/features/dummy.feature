Feature: Dummy

    Scenario: Resource is created
        When Resource is created:
        """
            apiVersion: v1
            kind: Secret
            metadata:
                name: host-operator-controller
        """
        Then Resource exists:
        """
            apiVersion: v1
            kind: Secret
            metadata:
                name: host-operator-controller
        """

