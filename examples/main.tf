terraform {
  required_providers {
    api7 = {
      source  = "registry.terraform.io/api7/api7"
      version = "0.1.0"
    }
  }
}

provider "api7" {
  endpoint         = "https://127.0.0.1:7443"
  api_key          = var.api7_api_key
  gateway_group_id = var.gateway_group_id
  insecure         = true
}

variable "api7_api_key" {
  type      = string
  sensitive = true
}

variable "gateway_group_id" {
  type    = string
  default = "default"
}
