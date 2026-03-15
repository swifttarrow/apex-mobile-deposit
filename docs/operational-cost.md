# Operational Cost Estimate

## Infrastructure (AWS us-east-1, ~1000 deposits/day)

| Component | Service | Monthly Cost |
|-----------|---------|-------------|
| Application | t3.small EC2 (1 instance) | ~$15 |
| Database | RDS PostgreSQL db.t3.micro | ~$25 |
| Image Storage | S3 (10GB + transfers) | ~$5 |
| Load Balancer | ALB | ~$20 |
| Monitoring | CloudWatch basic | ~$5 |
| **Total** | | **~$70/month** |

## Scaling to 10,000 deposits/day

| Component | Service | Monthly Cost |
|-----------|---------|-------------|
| Application | t3.medium × 2 behind ALB | ~$60 |
| Database | RDS PostgreSQL db.t3.small Multi-AZ | ~$100 |
| Cache | ElastiCache Redis (idempotency) | ~$25 |
| Image Storage | S3 (100GB) | ~$25 |
| Load Balancer | ALB | ~$25 |
| Monitoring | CloudWatch + X-Ray | ~$30 |
| **Total** | | **~$265/month** |

## Vendor Integration Costs

- Check vendor API: typically $0.01–$0.05 per check submitted
- At 1,000 deposits/day: $10–$50/day = $300–$1,500/month

## Development Costs

- Initial build: ~2 weeks of engineering time
- Ongoing maintenance: ~0.2 FTE (monitoring, incidents, compliance)

## Notes

- **Current demo:** Single binary (`bin/checkstream`) with embedded SQLite; no cloud infrastructure required for local dev or recording. Run with `make dev` (or `make build` then run the binary).
- SQLite used in this demo eliminates database cost for development/testing; see [DL-001](decision_log.md#dl-001-sqlite-over-postgresql).
- Real production requires PostgreSQL or Aurora for multi-node deployment.
- Settlement file transmission to bank may have per-file fees from the financial institution.
