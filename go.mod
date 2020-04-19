module github.com/itsthewall/thewall

go 1.13

replace github.com/DusanKasan/parsemail => github.com/itsthewall/parsemail v1.0.3

require (
	github.com/DusanKasan/parsemail v1.0.1
	github.com/gomarkdown/markdown v0.0.0-20200316172748-fd1f3374857d
	github.com/lib/pq v1.3.0
	github.com/sendgrid/rest v2.4.1+incompatible //indirect
	github.com/sendgrid/sendgrid-go v3.5.0+incompatible 
)
