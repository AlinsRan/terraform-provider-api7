resource "api7_route" "get_anything" {
  name       = "get-anything"
  service_id = api7_service.httpbin.id
  paths      = ["/anything/*"]
  methods    = ["GET"]
}
