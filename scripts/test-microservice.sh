#!/bin/bash
set -e

echo "Building microservice application..."
docker build -f Dockerfile.microservice -t stateledger-microservice:latest .

echo "Starting Kubernetes minikube..."
minikube start --memory 4096 --cpus 2

echo "Loading image into minikube..."
minikube image load stateledger-microservice:latest

echo "Applying deployment..."
kubectl apply -f deployments/k8s/microservice-deployment.yaml

echo "Waiting for deployment to be ready..."
kubectl rollout status deployment/microservice-app -n default --timeout=120s

echo "Port forwarding..."
kubectl port-forward service/microservice-app 8080:8080 &

sleep 2

echo ""
echo "=== Microservice Testing ==="
echo ""

# Test health check
echo "1. Testing health endpoint..."
curl -s http://localhost:8080/health | jq .

# Register users
echo ""
echo "2. Registering users..."
USER1=$(curl -s -X POST http://localhost:8080/users/register \
  -H "Content-Type: application/json" \
  -d '{
    "id": "user-001",
    "name": "Alice Johnson",
    "email": "alice@example.com"
  }' | jq .)
echo "$USER1"

USER2=$(curl -s -X POST http://localhost:8080/users/register \
  -H "Content-Type: application/json" \
  -d '{
    "id": "user-002",
    "name": "Bob Smith",
    "email": "bob@example.com"
  }' | jq .)
echo "$USER2"

# User login
echo ""
echo "3. User login..."
curl -s -X POST http://localhost:8080/users/user-001/login \
  -H "Content-Type: application/json" | jq .

# Create orders
echo ""
echo "4. Creating orders..."
ORDER1=$(curl -s -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{
    "id": "order-001",
    "user_id": "user-001",
    "amount": 299.99,
    "status": "pending"
  }' | jq .)
echo "$ORDER1"

ORDER2=$(curl -s -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{
    "id": "order-002",
    "user_id": "user-002",
    "amount": 599.99,
    "status": "pending"
  }' | jq .)
echo "$ORDER2"

# Process payment
echo ""
echo "5. Processing payment..."
curl -s -X POST http://localhost:8080/payments \
  -H "Content-Type: application/json" \
  -d '{
    "order_id": "order-001",
    "amount": 299.99
  }' | jq .

# Ship order
echo ""
echo "6. Shipping order..."
curl -s -X POST http://localhost:8080/orders/order-001/ship \
  -H "Content-Type: application/json" | jq .

# Get order details
echo ""
echo "7. Getting order details..."
curl -s http://localhost:8080/orders/order-001 | jq .

# List events
echo ""
echo "8. Listing all events..."
curl -s "http://localhost:8080/events?limit=20" | jq '.data | {count, records_sample: .records[0:3]}'

# Get user audit trail
echo ""
echo "9. User audit trail..."
curl -s http://localhost:8080/audit/user/user-001 | jq '.data | {user_id, count}'

# Get order audit trail
echo ""
echo "10. Order audit trail..."
curl -s http://localhost:8080/audit/order/order-001 | jq '.data | {order_id, count}'

# Get metrics
echo ""
echo "11. Microservice metrics..."
curl -s http://localhost:8080/metrics | jq '.data'

# User logout
echo ""
echo "12. User logout..."
curl -s -X POST http://localhost:8080/users/user-001/logout \
  -H "Content-Type: application/json" | jq .

echo ""
echo "=== Test Complete ==="
echo ""
echo "Microservice running on http://localhost:8080"
echo "Kubernetes pods: $(kubectl get pods -l app=microservice-app -o name)"
echo ""
echo "To stop:"
echo "  kubectl delete deployment microservice-app"
echo "  minikube stop"
