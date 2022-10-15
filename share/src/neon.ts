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
}

/**
 * Request interface
 *
 * @interface Request
 */
export interface Request {
  /**
   * Returns the request method.
   */
  method(): string;

  /**
   * Returns the request protocol.
   */
  proto(): string;

  /**
   * Returns the request protocol major version.
   */
  protoMajor(): number;

  /**
   * Returns the request protocol minor version.
   */
  protoMinor(): number;

  /**
   * Returns the request remote address.
   */
  remoteAddr(): string;

  /**
   * Returns the request host.
   */
  host(): string;

  /**
   * Returns the request path.
   */
  path(): string;

  /**
   * Returns the request query parameters.
   */
  query(): Record<string, string[]>

  /**
   * Returns the request headers.
   */
  headers(): Record<string, string[]>;

  /**
   * Returns the request state.
   */
  state(): Record<string, Resource>;
}

/**
 * Server interface
 *
 * @interface Response
 */
export interface Response {
  /**
   * Renders the response to the client.
   *
   * @param root the root content
   * @param status the response status code
   */
  render(root: string, status: number): void;

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
   * Sets the page title
   *
   * @param title the title
   */
  setTitle(title: string): void;

  /**
   * Sets a page metadata
   *
   * @param id the meta id
   * @param attributes the meta attributes
   */
  setMeta(id: string, attributes: Record<string, string>): void;

  /**
   * Sets a page link.
   *
   * @param id the link id
   * @param attributes the link attributes
   */
  setLink(id: string, attributes: Record<string, string>): void;

  /**
   * Sets a page script.
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
