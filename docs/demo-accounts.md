# Demo Accounts

Run the dev seed script before using these accounts:

```sh
scripts/populate_test_data.sh dev
```

## Local Logins

```text
System admin
Username: admin@example.com
Password: dev-password

Individual consumer
Username: developer@example.com
Password: dev-password

Corporate admin
Username: corp-admin@example.com
Password: dev-password
Company: Acme Creative Studio
```

## Corporate Signup Test

In the login dialog, switch to signup, choose `Corporate user`, and enter:

```text
Company name: Acme Creative Studio
```

The backend attaches the new user as a `corporate_member` by setting `sys_users.company_id` to the seeded company. Corporate admins can open the `Company Admin` view to see members, shared credit usage, and model usage distribution.

## Development Note

Seeded passwords use `dev-password` and are stored in `sys_users.password_hash` for local development. Production password storage should use a salted adaptive hash such as bcrypt or Argon2.
