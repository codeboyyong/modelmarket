  The live Docker demo could not run locally because Docker Desktop/Colima is not running. Once Docker is available, run:

  docker compose up -d --build
  scripts/init_db.sh dev
  scripts/populate_test_data.sh dev
  scripts/demo-smoke.sh dev
