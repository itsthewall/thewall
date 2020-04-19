# The Wall

## Development instructions

- run the binary once, it will run all the migrations and set up the DB for you.
- to put some test data in the DB run `$ psql <DBNAME> < sql/seed.sql`
- Put sendgrid API key in enviroment:
```bash
export SENDGRID_API_KEY='<KEY>'
export SENDGRID_TEMPLATE_ID='<TEMPLATE ID>'
```