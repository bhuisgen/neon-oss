(() => {
  // src/index.ts
  var root = `
<main>
  <div class="section">
    <div class="container">
      <div class="status">
        <img class="logo" src="/logo.png" alt="neon logo" />
        <p>Version ${server.version()}</p>
      </div>
    </div>
  </div>
</main>
`;
  response.render(root, 200);
  response.setTitle("Neon");
  response.setMeta("name", { "itemprop": "name", "content": "Neon" });
  response.setMeta("description", { "name": "description", "content": "Neon Status App" });
  response.setMeta("copyright", { "name": "copyright", "content": "Boris HUISGEN" });
  response.setMeta("generator", { "name": "generator", "content": `neon (engine='server', date='${new Date().toUTCString()}')` });
  response.setLink("canonical", { "rel": "canonical", "href": `http://${server.addr()}:${server.port()}${request.path()}` });
  response.setScript("schema-json-ld", {
    "type": "application/ld+json",
    "children": JSON.stringify({
      "@context": "http://schema.org",
      "@type": "WebSite",
      url: `http://${server.addr()}:${server.port()}${request.path()}`,
      name: "Neon",
      inLanguage: "en-US",
      description: "Neon Status App",
      keywords: "Neon",
      copyrightYear: new Date().getFullYear(),
      copyrightHolder: {
        "@type": "Person",
        name: "Boris HUISGEN"
      }
    })
  });
})();
