
## PostgreSQL CSV Backup & Restore Scripts
A pair of lightweight Python scripts to back up a PostgreSQL database by exporting each table into its own .csv file, archiving them into a single .zip file, and providing a seamless way to restore that data later.

Features
Zero CLI Dependency: Does not require system tools like pg_dump or pg_restore installed on your machine.

High Performance: Uses PostgreSQL's native COPY streaming protocols via psycopg2 for incredibly fast data transfers.

Smart Deletions: The restore script uses TRUNCATE ... CASCADE and resets auto-incrementing primary keys (RESTART IDENTITY) to guarantee a clean overwrite without foreign key clashes.

Prerequisites
Python 3.x installed.

PostgreSQL Adapter: You need the psycopg2 library to allow Python to talk to your database. Install it via pip:

Bash
pip install psycopg2-binary
Configuration
Before running either script, open backup_db.py and restore_db.py in your text editor and update the DB_CONFIG dictionary with your actual PostgreSQL credentials:

Python
DB_CONFIG = {
    "dbname": "your_database_name",  # The name of your database
    "user": "your_username",        # PostgreSQL username
    "password": "your_password",    # PostgreSQL password
    "host": "localhost",            # Database host server
    "port": "5432"                  # Default PostgreSQL port
}
How to Use
1. Backing Up the Database
Run the backup script from your terminal:

Bash
python backup_db.py
What it does:

Connects to the database and scans for all user-created tables in the public schema.

Generates a temporary .csv file for each table (with column headers included).

Compresses all CSVs into a timestamped file named db_backup_YYYYMMDD_HHMMSS.zip.

Automatically cleans up the temporary CSV folder.

2. Restoring the Database
⚠️ Critical Warning: The restore process uses TRUNCATE, meaning it deletes all existing data in the target tables before replacing it with the backup. Use with caution on production databases.

Additionally, this script restores data only. The destination database must already have the tables and schemas created.

Open restore_db.py and update the ZIP_TO_RESTORE variable near the top of the file with your specific backup filename:

Python
ZIP_TO_RESTORE = "db_backup_20260711_160000.zip"  # Replace with your actual file
Run the restore script:

Bash
python restore_db.py
What it does:

Unzips the designated backup file into a temporary folder.

Iterates through each CSV file, mapping columns automatically based on the CSV headers.

Wipes the target table cleanly while resetting primary key counters.

Streams the data from the CSV back into the table.

Commits the transaction if everything succeeds, or rolls back entirely if an error occurs.

Cleans up the temporary files.

can you put the readme into a md file

Python
readme_content = """# PostgreSQL CSV Backup & Restore Scripts

A pair of lightweight Python scripts to back up a PostgreSQL database by exporting each table into its own `.csv` file, archiving them into a single `.zip` file, and providing a seamless way to restore that data later.

## Features
* **Zero CLI Dependency:** Does not require system tools like `pg_dump` or `pg_restore` installed on your machine.
* **High Performance:** Uses PostgreSQL's native `COPY` streaming protocols via `psycopg2` for incredibly fast data transfers.
* **Smart Deletions:** The restore script uses `TRUNCATE ... CASCADE` and resets auto-incrementing primary keys (`RESTART IDENTITY`) to guarantee a clean overwrite without foreign key clashes.

---

## Prerequisites

1. **Python 3.x** installed.
2. **PostgreSQL Adapter:** You need the `psycopg2` library to allow Python to talk to your database. Install it via pip:

```bash
pip install psycopg2-binary
Configuration
Before running either script, open backup_db.py and restore_db.py in your text editor and update the DB_CONFIG dictionary with your actual PostgreSQL credentials:

Python
DB_CONFIG = {
    "dbname": "your_database_name",  # The name of your database
    "user": "your_username",        # PostgreSQL username
    "password": "your_password",    # PostgreSQL password
    "host": "localhost",            # Database host server
    "port": "5432"                  # Default PostgreSQL port
}
How to Use
1. Backing Up the Database
Run the backup script from your terminal:

Bash
python backup_db.py
What it does:

Connects to the database and scans for all user-created tables in the public schema.

Generates a temporary .csv file for each table (with column headers included).

Compresses all CSVs into a timestamped file named db_backup_YYYYMMDD_HHMMSS.zip.

Automatically cleans up the temporary CSV folder.

2. Restoring the Database
⚠️ Critical Warning: The restore process uses TRUNCATE, meaning it deletes all existing data in the target tables before replacing it with the backup. Use with caution on production databases.

Additionally, this script restores data only. The destination database must already have the tables and schemas created.

Open restore_db.py and update the ZIP_TO_RESTORE variable near the top of the file with your specific backup filename:

Python
ZIP_TO_RESTORE = "db_backup_20260711_160000.zip"  # Replace with your actual file
Run the restore script:

Bash
python restore_db.py
What it does:

Unzips the designated backup file into a temporary folder.

Iterates through each CSV file, mapping columns automatically based on the CSV headers.

Wipes the target table cleanly while resetting primary key counters.

Streams the data from the CSV back into the table.

Commits the transaction if everything succeeds, or rolls back entirely if an error occurs.

Cleans up the temporary files.
"""

with open("README.md", "w", encoding="utf-8") as f:
f.write(readme_content)

print("File generated successfully.")


```text?code_stdout&code_event_index=1
File generated successfully.

Your Markdown file is ready
MD icon
README
 MD 
Open

I have compiled the instructions, setup steps, and configuration documentation directly into a README.md file for your project directory.




Gemini is AI and can make mistakes.

Export Chat
# PostgreSQL CSV Backup & Restore Scripts

A pair of lightweight Python scripts to back up a PostgreSQL database by exporting each table into its own `.csv` file, archiving them into a single `.zip` file, and providing a seamless way to restore that data later.

## Features
* **Zero CLI Dependency:** Does not require system tools like `pg_dump` or `pg_restore` installed on your machine.
* **High Performance:** Uses PostgreSQL's native `COPY` streaming protocols via `psycopg2` for incredibly fast data transfers.
* **Smart Deletions:** The restore script uses `TRUNCATE ... CASCADE` and resets auto-incrementing primary keys (`RESTART IDENTITY`) to guarantee a clean overwrite without foreign key clashes.

---

## Prerequisites

1. **Python 3.x** installed.
2. **PostgreSQL Adapter:** You need the `psycopg2` library to allow Python to talk to your database. Install it via pip:

```bash
pip install psycopg2-binary
```

---

## Configuration

Before running either script, open `backup_db.py` and `restore_db.py` in your text editor and update the `DB_CONFIG` dictionary with your actual PostgreSQL credentials:

```python
DB_CONFIG = {
    "dbname": "your_database_name",  # The name of your database
    "user": "your_username",        # PostgreSQL username
    "password": "your_password",    # PostgreSQL password
    "host": "localhost",            # Database host server
    "port": "5432"                  # Default PostgreSQL port
}
```

---

## How to Use

### 1. Backing Up the Database

Run the backup script from your terminal:

```bash
python backup_db.py
```

**What it does:**
1. Connects to the database and scans for all user-created tables in the `public` schema.
2. Generates a temporary `.csv` file for each table (with column headers included).
3. Compresses all CSVs into a timestamped file named `db_backup_YYYYMMDD_HHMMSS.zip`.
4. Automatically cleans up the temporary CSV folder.

---

### 2. Restoring the Database

> ⚠️ **Critical Warning:** The restore process uses `TRUNCATE`, meaning it **deletes all existing data** in the target tables before replacing it with the backup. Use with caution on production databases.
> 
> Additionally, this script restores **data only**. The destination database must already have the tables and schemas created.

1. Open `restore_db.py` and update the `ZIP_TO_RESTORE` variable near the top of the file with your specific backup filename:
   ```python
   ZIP_TO_RESTORE = "db_backup_20260711_160000.zip"  # Replace with your actual file
   ```
2. Run the restore script:
   ```bash
   python restore_db.py
   ```

**What it does:**
1. Unzips the designated backup file into a temporary folder.
2. Iterates through each CSV file, mapping columns automatically based on the CSV headers.
3. Wipes the target table cleanly while resetting primary key counters.
4. Streams the data from the CSV back into the table.
5. Commits the transaction if everything succeeds, or rolls back entirely if an error occurs.
6. Cleans up the temporary files.
README.md
Displaying README.md.