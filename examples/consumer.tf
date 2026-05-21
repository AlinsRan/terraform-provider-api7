resource "api7_consumer" "alice" {
  username = "alice"
  desc     = "Test consumer created by Terraform"
  # Note: API7 EE requires plugins/credentials to be managed via
  # the credential sub-resource, not directly on the consumer.
}

output "consumer_username" {
  value = api7_consumer.alice.username
}
