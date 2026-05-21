resource "api7_service" "httpbin" {
  name = "httpbin"
  desc = "HTTPBin 测试服务"

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
