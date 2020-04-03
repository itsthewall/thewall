



var darkMode = false;
var element = null;

//https://www.geeksforgeeks.org/how-to-load-css-files-using-javascript/
function darkmode() {
    if (darkMode) {
        element.remove()
        darkMode = false;
        return;
    }

    var head = document.getElementsByTagName('HEAD')[0];  
  
    // Create new link Element 
    var link = document.createElement('link'); 

    // set the attributes for link element  
    link.rel = 'stylesheet';  
    
    link.type = 'text/css'; 
    
    link.href = '/static/darkmode.css';  

    // Append link element to HTML head 
    element = head.appendChild(link);
    darkMode = true;
}