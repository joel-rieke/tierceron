//library vault_vals.json_to_object;

import 'dart:html';
import 'dart:core';
import 'dart:async';
import 'dart:convert';
//import 'dart:io';
//import 'package:dson/dson.dart';

import 'package:http/browser_client.dart';
import 'package:angular/angular.dart';
import 'package:angular_forms/angular_forms.dart';
import 'package:angular_router/angular_router.dart';

//import 'dart-ext:C:/Users/Sara.wille/workspace/go/src/bitbucket.org/dexterchaney/whoville/webapi/rpc/apinator';
import '../routes.dart';

// Displayed when vault has not yet been created,w
//part 'json_to_object.g.dart';
//part 'extend_serializables.g.dart';
@Component(
  selector: 'vault-vals',
  styleUrls: ['active_sessions_component.css'],
  templateUrl: 'active_sessions_component.html',
  directives: [coreDirectives, routerDirectives,formDirectives],
  pipes: const [SlicePipe],
  providers: const [ClassProvider(Routes), commonPipes]
)

@Injectable()
class ActiveSessionsComponent implements OnInit{
  final Routes routes;
  final Router _router;
  ActiveSessionsComponent(this.routes, this._router);

  String _baseHref;
  String _token;
  String valQuery = '/graphql?query={envs{name, providers{name, sessions{User, LastLogIn}}}}';
  String envNameQuery = '/graphql?query={envs{name}}';
  String provNameQuery = '/graphql?query={envs{providers{name}}}';
  Start()async {
    await GetSessions();
    GetEnvNames();
    GetProvNames();
  }

  Future<void> ngOnInit() async {
    _baseHref = window.location.origin;
    _token = "Bearer " + window.localStorage['Token'];
    Start();
  }
  SignOut()async{
    //sign out and redirect to login page
    bool isSealed;
    final  String _logInEndpoint = window.location.origin + '/twirp/viewpoint.whoville.apinator.EnterpriseServiceBroker/GetStatus'; 
    HttpRequest request = new HttpRequest();
    request.onLoadEnd.listen((_) {
      Map<String, dynamic> response = json.decode(request.responseText);
      // Convert null values to false; Extract vault status
      isSealed = response['sealed'] == null ? false : response['sealed'] as bool;
    
      //print("sealed: " + isSealed.toString());
      // Vault seeded, user needs to login and recieve token. Vault possibly needs to be unsealed
      print("logout");
      window.localStorage.clear();
      _router.navigate(routes.login.toUrl(), NavigationParams(queryParameters: {'sealed': isSealed.toString()}, reload: true));
    });
    request.open('POST', _logInEndpoint);
    request.setRequestHeader('Content-Type', 'application/json');
    await request.send('{}');
  }
  
  ConfigBrowser(){
    //redirect to configuration browser
    print("values");
    _router.navigate(routes.values.toUrl(), NavigationParams(reload: true));
  }
  GetSessions() async {
    var client = new BrowserClient();
    var response =
        await client.get(valQuery, headers: {'Authorization': _token});
      if (response.statusCode == 401) { // Unauthorized, redirect to login page
        window.localStorage.remove("Token");
        window.location.href = routes.login.toUrl();
      }
      Map activeSessions = json.decode(response.body);

      List users = [];
      List logins = [];
      Map data = activeSessions['data'];
      List envs = data['envs'];
      if(envs != null){

        for(var i = 0; i < envs.length; i++){
        Map envMap = envs[i];
        List providers = envMap['providers'];
        if(providers != null){

          for(var i = 0; i < providers.length; i++){
            Map providerMap = providers[i];
            List sessionList = providerMap['sessions'];
            if(sessionList != null){
              // Add header
              for(var i = 0; i < sessionList.length; i++){
                Map userMap = sessionList[i];

                String name = userMap['user'];
                int loginTime = userMap['lastLogIn'];
                if(name != null){
                  //only adds values if they have a corresponding key
                  users.add(name);
                  logins.add(loginTime);
                }
                else{
                  print("name is null");
                }
              }
            }
          
            }
          }
        }
      }
    
      // Add a header
      var userList = querySelector('#users');
      userList.children.clear();
      for(var i = 0; i < users.length; i++){
        var newUser = new LIElement();
        newUser.text = users[i];
        newUser.classes.add(userList.classes.last);
        userList.children.insert(userList.children.length, newUser);
      }

      var loginList = querySelector('#logins');
      loginList.children.clear();
      for(var i = 0; i < logins.length; i++){
        var newLogin = new LIElement();
        newLogin.text = logins[i];
        newLogin.classes.add(loginList.classes.last);
        loginList.children.insert(loginList.children.length, newLogin);
      }


  }
  GetEnvNames() async{
    var client = new BrowserClient();
    var response =
      await client.get(envNameQuery, headers: {'Authorization': _token});
    if (response.statusCode == 401) { // Unauthorized, redirect to login page
        window.localStorage.remove("Token");
        window.location.href = routes.login.toUrl();
    }
    Map vaultVals = json.decode(response.body);
    Map data = vaultVals['data'];
    List envMaps = data['envs'];
    List envNames = [];

    for(var i=0; i<envMaps.length; i++){
      var envMap = envMaps[i];
      var envName = envMap['name'];
      envNames.add(envName);
    }

    var envList = querySelector('#environments');
    envList.children.clear();
    var newEnv = new OptionElement();
    newEnv.text = 'All Environments';
    newEnv.value = '';
    envList.children.add(newEnv);
    for(var i = 0; i < envNames.length; i++){
      var newEnv = new OptionElement();
      newEnv.text = envNames[i];
      newEnv.value = envNames[i];
      envList.children.add(newEnv);
    }
  }
  GetProvNames()async{

    var client = new BrowserClient();
    var response =
      await client.get(provNameQuery, headers: {'Authorization': _token});
    if (response.statusCode == 401) { // Unauthorized, redirect to login page
        window.localStorage.remove("Token");        
        window.location.href = routes.login.toUrl();
    }

    Map vaultVals = json.decode(response.body);
    Map data = vaultVals['data'];
    List envList = data['envs'];
    Set provNames = Set();

    for(var i=0; i<envList.length; i++){
      var envMap = envList[i];
      var provList = envMap['providers'];
      for(var i=0;i<provList.length; i++){
        var provMap = provList[i];
        var provName = provMap['name'];
        provNames.add(provName);
      }
    }

    var provList = querySelector('#providers');
    provList.children.clear();
    var newProv = new OptionElement();
    newProv.text = 'All Auth Providers';
    newProv.value = '';
    provList.children.add(newProv);
    for(var i = 0; i < provNames.length; i++){
      var newProv = new OptionElement();
      newProv.text = provNames.elementAt(i);
      newProv.value = provNames.elementAt(i);
      provList.children.add(newProv);
      
    }

  }
  EditQuery(String selected, String structName, String nameKey, String query){
    var isFilled = query.contains(nameKey);
    if(isFilled){
      var startIndex = query.indexOf(nameKey);
      var endIndex = query.indexOf(')', startIndex);
      // Check whether this is the last argument
      var commaIndex = query.indexOf(',', startIndex);
      if (commaIndex < endIndex) {
        endIndex = commaIndex + 1;
      }
      var startString = query.substring(0, startIndex);
      var endString = query.substring(endIndex, query.length);
      query = startString + endString;
    }

    var hasArguments = query.contains(structName + '('); // Check if other arugments exist
    var otherArguments = '';
    if(hasArguments) {
      var startIndex = query.indexOf('(', query.indexOf(structName));
      var endIndex = query.indexOf(')', startIndex) + 1;
      otherArguments = query.substring(startIndex+1, endIndex - 1);
      otherArguments = otherArguments.trim();
      if (otherArguments.length > 0 && otherArguments[otherArguments.length - 1] ==  ',') {
        otherArguments = otherArguments.substring(0, otherArguments.length - 1);
      }
      var startString = query.substring(0, startIndex);
      var endString = query.substring(endIndex, query.length);
      query = startString + endString;
    }

    var index = query.indexOf(structName) + structName.length;
    var startString = query.substring(0, index);
    var endString = query.substring(index, query.length);
    if(selected.length > 0) {
      if (otherArguments.length > 0){
        query = startString + '(' + otherArguments + ', ' + nameKey+':\"'+ selected +'\")' + endString;
      } else {
        query = startString + '(' + nameKey+':\"'+ selected +'\")' + endString;

      }
    } else if (otherArguments.length > 0) {
      query = startString + '(' + otherArguments + ')' + endString;      
    } else {
      query = startString + endString;
    }
    return query;
  }
  SelectEnv(String env){

      ResetValQuery();
      ResetEnvNameQuery();
      ResetProvNameQuery();

    if(env != ''){
      valQuery = EditQuery(env, 'envs', 'envName', valQuery);
      provNameQuery = EditQuery(env, 'envs', 'envName', provNameQuery);
    }

    GetSessions();
    GetProvNames();

    (querySelector('#source') as SelectElement).selectedIndex = 0;
    (querySelector('#query') as InputElement).value = '';    
  }
  SelectProv(String prov){


      valQuery = RemoveQueryElement(valQuery, 'provName');
      valQuery = RemoveQueryElement(valQuery, 'fileName');

    if(prov != ''){
      valQuery = EditQuery(prov, 'providers', 'provName', valQuery);
    }
    GetSessions();

    (querySelector('#source') as SelectElement).selectedIndex = 0;
    (querySelector('#query') as InputElement).value = '';    
  }
  SelectFile(String file){

      valQuery = RemoveQueryElement(valQuery, 'fileName');

    if(file != ''){
      valQuery = EditQuery(file, 'files', 'fileName', valQuery);
    }

    GetSessions();

    (querySelector('#source') as SelectElement).selectedIndex = 0;
    (querySelector('#query') as InputElement).value = '';
  }
  SelectKey(String key){
    valQuery = EditQuery(key, 'values', 'keyName', valQuery);
    GetSessions();
  }
  SelectSource(String source) {
    valQuery = EditQuery(source, 'values', 'sourceName', valQuery);
    GetSessions();
  }
  ResetValQuery(){
    valQuery = _baseHref + '/graphql?query={envs{name, providers{name, sessions{User, LastLogIn}}}}';
  }
  ResetEnvNameQuery(){
    envNameQuery =  _baseHref + '/graphql?query={envs{name}}';
  }
  ResetProvNameQuery(){
    provNameQuery =  _baseHref + '/graphql?query={envs{providers{name}}}';
  }
  RemoveQueryElement(String query, String nameKey){
    var isFilled = query.contains(nameKey);

    if(isFilled){
      var startIndex = query.indexOf("(" + nameKey);
      var endIndex = query.indexOf(")", startIndex) + 1;
      var startString = query.substring(0, startIndex);
      var endString = query.substring(endIndex, query.length);
      query = startString + endString;
    }
    return query;
  }
}