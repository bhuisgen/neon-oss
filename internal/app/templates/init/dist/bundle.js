(()=>{var e=`
<main>
  <div class="section">
    <div class="container">
      <div class="status">
        <img class="logo" src="/logo.png" alt="neon logo">
        <p>Version ${server.version()}</p>
      </div>
    </div>
  </div>
</main>
`;serverResponse.render(e,200);serverResponse.setTitle("Neon");serverResponse.setMeta("name",new Map([["itemprop","name"],["content","Neon"]]));serverResponse.setMeta("description",new Map([["name","description"],["content","Neon Status App"]]));serverResponse.setMeta("copyright",new Map([["name","copyright"],["content","Boris HUISGEN"]]));serverResponse.setMeta("generator",new Map([["name","generator"],["content",`neon (engine='server', date='${new Date().toUTCString()}')`]]));serverResponse.setLink("canonical",new Map([["rel","canonical"],["href",`http://${server.addr()}:${server.port()}${serverRequest.path()}`]]));serverResponse.setScript("schema-json-ld",new Map([["type","application/ld+json"],["children",JSON.stringify({"@context":"http://schema.org","@type":"WebSite",url:`http://${server.addr()}:${server.port()}${serverRequest.path()}`,name:"Neon",inLanguage:"en-US",description:"Neon Status App",keywords:"Neon",copyrightYear:new Date().getFullYear(),copyrightHolder:{"@type":"Person",name:"Boris HUISGEN"}})]]));})();
