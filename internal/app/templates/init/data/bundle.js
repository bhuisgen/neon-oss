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
  server.render(root, 200);
  server.setTitle("Neon");
  server.setMeta("name", { "itemprop": "name", "content": "Neon" });
  server.setMeta("description", { "name": "description", "content": "Neon Status App" });
  server.setMeta("copyright", { "name": "copyright", "content": "Boris HUISGEN" });
  server.setMeta("generator", { "name": "generator", "content": `neon (engine='server', date='${new Date().toUTCString()}')` });
  server.setLink("canonical", { "rel": "canonical", "href": `http://${server.addr()}:${server.port()}${server.requestPath()}` });
  server.setScript("schema-json-ld", {
    "type": "application/ld+json",
    "children": JSON.stringify({
      "@context": "http://schema.org",
      "@type": "WebSite",
      url: `http://${server.addr()}:${server.port()}${server.requestPath()}`,
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
