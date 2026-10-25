 // Example of  Http request transformation from js  
 //
 
function execjs()
{
    // Auto login mantis ( from sso in reality )
    Remote_User = "administrator"

    // On Get login_page.php Mantis ==>  rewrite to Post login.php, with form-data to authenticate 
    if ( Remote_User !== "admin" && Method === "GET" && !Query_String ) {
        return ({     
            "Method": "POST",
            "Url":  Backend_URL.replace("_page",""),
            "Content-Type": "application/x-www-form-urlencoded",       // const password 
            "Body": "return=index.php&username=" + Remote_User + "&password=chdeul&perm_login=on"   
            // Content-length : automatique
        })
    } else  {
        // Normal proxified request
        // Login page for Admin or error login page ... 
        return ({ })     
    }
}

// Return to go
execjs()