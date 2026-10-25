 // Example of Http request callable from js  
 //
 // data = httpCall( method, url, token, body, cookie ) 
 //
 // set at return the variables  Set_Cookie,  Status  into  javascript
 //
 // Content-Type will be json if data starts by { or [ 
 //        else will be formdata  on  POST
 //        else text ( default )
 //
 // The url can be treat by Proxy itself, follows redirections, but no execjs ( no recursive js )
 //

function execjs()
{   
   // Use frontend url, not backend url , because strict controls of openproject !
   let json = httpCall("GET", Frontend_URL, "VARUcfBRltaH14PmwdVZ_AMd-87TLQqz", "", Cookie) 

   return { Data: json, Code : Status }
   // ou return json : données seules   
   // ou return code entier du status, sans données 
}

// Return to go
execjs()