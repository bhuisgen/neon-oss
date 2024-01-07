// Copyright 2022-2024 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package neon

import (
	_ "github.com/bhuisgen/neon/pkg/modules/listener/listeners/local"
	_ "github.com/bhuisgen/neon/pkg/modules/listener/listeners/redirect"
	_ "github.com/bhuisgen/neon/pkg/modules/listener/listeners/tls"

	_ "github.com/bhuisgen/neon/pkg/modules/server/middlewares/compress"
	_ "github.com/bhuisgen/neon/pkg/modules/server/middlewares/header"
	_ "github.com/bhuisgen/neon/pkg/modules/server/middlewares/logger"
	_ "github.com/bhuisgen/neon/pkg/modules/server/middlewares/rewrite"
	_ "github.com/bhuisgen/neon/pkg/modules/server/middlewares/static"

	_ "github.com/bhuisgen/neon/pkg/modules/server/handlers/app"
	_ "github.com/bhuisgen/neon/pkg/modules/server/handlers/file"
	_ "github.com/bhuisgen/neon/pkg/modules/server/handlers/robots"
	_ "github.com/bhuisgen/neon/pkg/modules/server/handlers/sitemap"

	_ "github.com/bhuisgen/neon/pkg/modules/fetcher/providers/rest"

	_ "github.com/bhuisgen/neon/pkg/modules/loader/parsers/json"
	_ "github.com/bhuisgen/neon/pkg/modules/loader/parsers/raw"
)
