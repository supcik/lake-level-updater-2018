# Lake Level Updater

This app is designed to run in the "App Engine" (Standard environment) of the "Google Cloud Service".
It fetches lake level information from the Internet (https://www.groupe-e.ch/fr/univers-groupe-e/niveau-lacs)
and updates a Firebase Realtime Database.

Note: The "Groupe-e" does not provide an API where we can get lake level information, but they aggreed that it was OK for me to get this information from their public web page. The risk is that this service
may no longer work if the Group-E decides to change the layout of their web page.