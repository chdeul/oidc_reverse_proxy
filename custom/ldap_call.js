// Example of Ldap request callable from js  
 //
 // data = ldapCall( url, bind, pasword, dn, search_pattern, attr ) 
 //
 // set at return a variable  Status ( Ok, NoContent ou InternalError : as http code )  into javascript
 //
 // The search is on children only, not deep
 //

function execjs()
{   
    // Pattern de recherche
    Remote_User= "a*"

     // Exemple of ldap call
    let data = ldapCall( "ldap://localhost:389",        // Ldap url
    "cn=admin,dc=ldap,dc=example,dc=org", "admin123",   // if auth required 
    "ou=users,dc=ldap,dc=example,dc=org",               // dn , root where search 
    "(&(objectClass=inetOrgPerson)(sn="+Remote_User+"))", // pattern
        "sn,userPassword") // attr list : returned
        
   // format data : multi-lines
   //     attr=valeur,attr=valeur,attr=valeur1:valeur2
   //     attr=valeur,attr=valeur,attr=valeur1:valeur2
   //     attr=valeur,attr=valeur,attr=valeur1:valeur2
   
   // ex: sn=antony,userPassword={SSHA}8Tx9HkNNvNqIlhhQHrbKPTT+h9ZHU8Wt
   //     sn=admin,userPassword={SSHA}suCkd0CLg0o9dmADdqJnULp1V8EeA0tH

   return { Data: data, Code : Status }
   // ou return json : data only   
   // ou return code entier du status, without data ( ok = 200 )
}

// Return to go
execjs() 
 