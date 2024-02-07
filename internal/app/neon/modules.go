package neon

import (
	_ "github.com/bhuisgen/neon/pkg/modules/store/storage/memory"

	_ "github.com/bhuisgen/neon/pkg/modules/fetcher/providers/file"
	_ "github.com/bhuisgen/neon/pkg/modules/fetcher/providers/rest"

	_ "github.com/bhuisgen/neon/pkg/modules/loader/parsers/json"
	_ "github.com/bhuisgen/neon/pkg/modules/loader/parsers/raw"

	_ "github.com/bhuisgen/neon/pkg/modules/server/listener/listeners/local"
	_ "github.com/bhuisgen/neon/pkg/modules/server/listener/listeners/redirect"
	_ "github.com/bhuisgen/neon/pkg/modules/server/listener/listeners/tls"

	_ "github.com/bhuisgen/neon/pkg/modules/server/site/middlewares/compress"
	_ "github.com/bhuisgen/neon/pkg/modules/server/site/middlewares/header"
	_ "github.com/bhuisgen/neon/pkg/modules/server/site/middlewares/logger"
	_ "github.com/bhuisgen/neon/pkg/modules/server/site/middlewares/rewrite"
	_ "github.com/bhuisgen/neon/pkg/modules/server/site/middlewares/static"

	_ "github.com/bhuisgen/neon/pkg/modules/server/site/handlers/app"
	_ "github.com/bhuisgen/neon/pkg/modules/server/site/handlers/file"
	_ "github.com/bhuisgen/neon/pkg/modules/server/site/handlers/robots"
	_ "github.com/bhuisgen/neon/pkg/modules/server/site/handlers/sitemap"
)
