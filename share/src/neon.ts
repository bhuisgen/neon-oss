// Copyright 2022 Boris HUISGEN. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

/**
 * Server interface
 *
 * @interface Server
 */
export interface Server {
  /**
   * Return the server address.
   */
  addr(): string;

  /**
   * Return the server port.
   */
  port(): number;

  /**
   * Returns the server version.
   */
  version(): string;

  /**
   * Returns the request method.
   */
  requestMethod(): string;

  /**
   * Returns the request protocol.
   */
  requestProto(): string;

  /**
   * Returns the request protocol major version.
   */
  requestProtoMajor(): number;

  /**
   * Returns the request protocol minor version.
   */
  requestProtoMinor(): number;

  /**
   * Returns the request remote address.
   */
  requestRemoteAddr(): string;

  /**
   * Returns the request host.
   */
  requestHost(): string;

  /**
   * Returns the request path.
   */
  requestPath(): string;

  /**
   * Returns the request query parameters.
   */
  requestQuery(): Record<string, string[]>

  /**
   * Returns the request headers.
   */
  requestHeaders(): Record<string, string[]>;

  /**
   * Returns the state.
   */
  state(): Record<string, Resource>;

  /**
   * Renders the page to client.
   *
   * @param html the HTML content
   * @param status the response status code
   */
  render(html: string, status: number): void;

  /**
   * Redirects the client to another URL.
   *
   * @param url the redirect URL
   * @param status the redirect status code
   */
  redirect(url: string, status: number): void;

  /**
   * Sets a response header
   *
   * @param key the key
   * @param value  the value
   */
  setHeader(key: string, value: string): void;

  /**
   * Sets the HTML page title
   *
   * @param title the title
   */
  setTitle(title: string): void;

  /**
   * Sets the HTML page metadata
   *
   * @param id the meta id
   * @param attributes the meta attributes
   */
  setMeta(id: string, attributes: Record<string, string>): void;

  /**
   * Sets the HTML page link.
   *
   * @param id the link id
   * @param attributes the link attributes
   */
  setLink(id: string, attributes: Record<string, string>): void;

  /**
   * Sets the HTML page script.
   *
   * @param id the script id
   * @param attributes the script attributes
   */
  setScript(id: string, attributes: Record<string, string>): void;
}

/**
 * Resource interface
 *
 * @interface Resource
 */
export interface Resource {
  /**
   * The resource loading state.
   */
  loading: boolean;

  /**
   * The resource error state.
   */
  error: string | null;

  /**
   * The resource response.
   */
  response: string | null;
}
