/**
 * Resource interface.
 *
 * @interface Resource
 */
export interface Resource {
  /**
   * The resource data.
   */
  data: string[];

  /**
   * The resource error.
   */
  error: string | null;
}

/**
 * Site interface.
 *
 * @interface Site
 */
interface Site {
  /**
   * Return the site name.
   *
   * @returns {string} The site name.
   */
  name(): string;

  /**
   * Returns the site listeners.
   *
   * @returns {string[]} The list of listeners.
   */
  listeners(): string[];

  /**
   * Return the site hosts.
   *
   * @returns {string[]} The list of hosts.
   */
  hosts(): string[];

  /**
   * Returns true if the site is the default site of the server.
   *
   * @returns {boolean} The default site flag.
   */
  isDefault(): boolean;
}

/**
 * Handler interface.
 *
 * @interface Handler
 */
interface Handler {
  /**
   * Returns the handler state.
   */
  state(): Record<string, Resource>;
}

/**
 * Request interface.
 *
 * @interface Request
 */
interface Request {
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
  query(): Record<string, string[]>;

  /**
   * Returns the request headers.
   */
  headers(): Record<string, string[]>;
}

/**
 * Response interface.
 *
 * @interface Response
 */
interface Response {
  /**
   * Renders the response to the client.
   *
   * @param content the content
   * @param status the response status code
   */
  render(content: string, status: number): void;

  /**
   * Redirects the client to another URL.
   *
   * @param url the redirect URL
   * @param status the redirect status code
   */
  redirect(url: string, status: number): void;

  /**
   * Sets a response header.
   *
   * @param key the key
   * @param value  the value
   */
  setHeader(key: string, value: string): void;

  /**
   * Sets the page title.
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
  setMeta(id: string, attributes: Map<string, string>): void;

  /**
   * Sets a page link.
   *
   * @param id the link id
   * @param attributes the link attributes
   */
  setLink(id: string, attributes: Map<string, string>): void;

  /**
   * Sets a page script.
   *
   * @param id the script id
   * @param attributes the script attributes
   */
  setScript(id: string, attributes: Map<string, string>): void;
}

/**
 * Server interface.
 *
 * @interface Server
 */
export interface Server {
  /**
   * The site object.
   */
  site: Site;

  /**
   * The handler object.
   */
  handler: Handler;

  /**
   * The request object.
   */
  request: Request;

  /**
   * The response object.
   */
  response: Response;
}
