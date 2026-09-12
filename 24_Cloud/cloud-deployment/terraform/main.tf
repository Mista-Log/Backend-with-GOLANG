# A minimal, illustrative sketch — deliberately not a complete, applyable
# configuration (that would need a VPC, subnets, an ECR repo, IAM roles,
# and a load balancer defined too). This shows the SHAPE of declaring an
# ECS Fargate service for cloudapp; see the guide's Terraform section for
# the plan/apply workflow this file is meant to support.

resource "aws_ecs_cluster" "main" {
  name = "cloudapp-cluster"
}

resource "aws_ecs_task_definition" "cloudapp" {
  family                   = "cloudapp"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = "256"
  memory                   = "512"

  container_definitions = jsonencode([{
    name  = "cloudapp"
    image = "cloudapp:latest" # in practice: an ECR image URI, pushed by CI/CD
    portMappings = [{ containerPort = 8080 }]
    healthCheck = {
      command  = ["CMD-SHELL", "wget -qO- http://localhost:8080/healthz || exit 1"]
      interval = 10
      timeout  = 3
      retries  = 3
    }
  }])
}

resource "aws_ecs_service" "cloudapp" {
  name            = "cloudapp"
  cluster         = aws_ecs_cluster.main.id
  task_definition = aws_ecs_task_definition.cloudapp.arn
  desired_count   = 3
  launch_type     = "FARGATE"

  network_configuration {
    subnets = [] # real subnet IDs go here
  }
}
