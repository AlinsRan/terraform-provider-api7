resource "api7_service" "example" {
  name = "httpbin"
  desc = "HTTPBin service"

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
