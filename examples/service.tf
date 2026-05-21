resource "api7_service" "httpbin" {
  name = "httpbin"
  desc = "HTTPBin service for testing"

  upstream = {
    nodes = [
      {
        host   = "httpbin.org"
        port   = 80
        weight = 100
      }
    ]
    scheme = "http"
    type   = "roundrobin"
  }
}

output "service_id" {
  value = api7_service.httpbin.id
}
