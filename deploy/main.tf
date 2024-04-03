terraform {
  backend "s3" {
    bucket = "testgrid-tf-state"
    key    = "testgrid-prod/terraform.tfstate"
    region = "us-east-1"
  }
}

provider "aws" {
  region = "us-east-1"
}

resource "equinix_metal_device" "device" {
  count = var.device_count

  project_id       = var.project_id
  metro            = var.region[count.index]
  hostname         = "${var.tg_hostname}-${count.index}"
  billing_cycle    = "hourly"
  operating_system = var.tg_os
  plan             = var.instance_type
  user_data        = file("tg-script.sh")
  hardware_reservation_id = length(var.reservation_ids) > count.index ? var.reservation_ids[count.index] : ""
}

output "device_ids" {
  value = equinix_metal_device.device.*.id
}

