// String Variables incoming
// - Remote_User,   if sso auth 
// - Remote_IP,   
// - Method,
// - Frontend_URL,  asked url
// - Backend_URL,   proxified url
// - Query_String,  if any
// - Authorization, if any
// - Cookie,        if any 
// - Content_Type,  
// - Body,          if application/json ou application/x-www-form-urlencoded

function execjs()
{   
    if ( Method === "GET" )
    {
        let incoming = "<html><body><h3>ExecJs variables :</h3><p>"
        incoming += "Remote_User: " + Remote_User + "<br>"
        incoming += "Remote_IP: " + Remote_IP + "<br>"
        incoming += "Method: " + Method + "<br>"
        incoming += "Frontend_URL: " + Frontend_URL + "<br>"
        incoming += "Backend_URL: " + Backend_URL + "<br>"
        incoming += "Query_String: " + Query_String + "<br>"
        incoming += "Authorization: " + Authorization + "<br>"
        incoming += "Cookie: " + Cookie + "<br>"
        incoming += "Content_Type: " + Content_Type + "<br>"    
        incoming += "Body: " + Body + "<br>"
        incoming += "</p></body></html>"

        // retourne des datas directement
        return ({ "Data" : incoming, "Code" : 200 })
    }
    else
    {
        // Sinon transmet la Requete proxifiee et son résultat
        return ({ })        
    }
}

// Return to go
execjs()