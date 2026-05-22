resource "api7_route" "get_anything" {
  name       = "get-anything"
  service_id = api7_service.httpbin.id
  paths      = ["/anything/*"]
  methods    = ["GET"]
}


resource "api7_route" "get" {
  name       = "get"
  service_id = api7_service.httpbin.id
  paths      = ["/get"]
  methods    = ["GET"]
  plugins = jsonencode({
    "key-auth" = {}
  })
}

resource "api7_route" "headers" {
  name       = "headers"
  service_id = api7_service.httpbin.id
  paths      = ["/headers"]
  methods    = ["GET", "POST"]
}

