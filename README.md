
# vue intégration

- self maded <embed> html tag, includes .vue,  managed internally
- js library included into index.html, from resources, already downloaded



Authentification  (/auth)  login & password
- data stocké dans users.csv (fichier)
- login -> JWT Token avec identifiant,profil (jwttoken)
- JWT renvoyé en cookie dans chaque requete REST axios (app.vue)
- chaque requete REST : cherche JWT, vérifie si authentifié et profil  (RestApi)

Static ( /ressources /vues/ )
- Html,js,css,img,vue : static, requetes sans Authentification  (main.go)

Renouvellement : changement mot de passe
- requete renew avec identifiant, mail, sans password
- envoi un email avec jeton JWT Profil "Renouvellement", avec lien
- sur appel lien renouvellement : decode, vérifie jeton, modifie  password


glue

in
  ldap
  sql
  rest webhooks

out
 x ldap
  sql
  rest webhooks

xsso
x  openidconnect

gui

x  login form
  droits correspondances
  create user form
  remove user batch
  create and remove project 
  logout chains
  batch incoherence, password expiration
  ldap edit

front apache proxy
  mod_openid
