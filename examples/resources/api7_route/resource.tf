resource "api7_route" "example" {
  name       = "get-anything"
  service_id = api7_service.example.id
  paths      = ["/anything/*"]
  methods    = ["GET"]
}
