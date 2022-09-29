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
   * The server request URL.
   */
  url: string;

  /**
   * The server state.
   */
  state: Record<string, ServerResource>;

  /**
   * Renders a page.
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
 * Server resource interface
 *
 * @interface ServerResource
 */
export interface ServerResource {
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
