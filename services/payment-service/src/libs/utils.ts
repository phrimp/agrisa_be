const generateRandomString = (length: number): string => {
  const characters =
    'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
  let result = '';
  const charactersLength = characters.length;
  for (let i = 0; i < length; i++) {
    result += characters.charAt(Math.floor(Math.random() * charactersLength));
  }
  return result;
};

const toCamelCase = (str: string): string =>
  str.replace(/_([a-z])/g, (_, letter: string) => letter.toUpperCase());

const toSnakeCase = (str: string): string =>
  str.replace(/[A-Z]/g, (letter: string) => `_${letter.toLowerCase()}`);

const transformKeys = (obj: any, transform: (key: string) => string): any => {
  if (obj === null || typeof obj !== 'object') return obj;
  if (Array.isArray(obj))
    // eslint-disable-next-line @typescript-eslint/no-unsafe-return
    return obj.map((item: any) => transformKeys(item, transform));
  const transformed: any = {};
  for (const key in obj) {
    if (Object.prototype.hasOwnProperty.call(obj, key)) {
      // eslint-disable-next-line @typescript-eslint/no-unsafe-assignment, @typescript-eslint/no-unsafe-member-access
      transformed[transform(key)] = transformKeys(obj[key], transform);
    }
  }
  return transformed;
};

export { generateRandomString, toCamelCase, toSnakeCase, transformKeys };
