const generateRandomString = (length: number = 10): string => {
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
    return obj.map((item: any) => transformKeys(item, transform));
  const transformed: any = {};
  for (const key in obj) {
    if (Object.prototype.hasOwnProperty.call(obj, key)) {
      transformed[transform(key)] = transformKeys(obj[key], transform);
    }
  }
  return transformed;
};

const checkPermissions = (
  permissions: string[],
  required: string[],
): boolean => {
  return required.every((perm) => permissions.includes(perm));
};

const generateReferenceId = (prefix: string = 'payout'): string => {
  const timestamp = Date.now();
  const random = Math.floor(Math.random() * 1000)
    .toString()
    .padStart(3, '0');
  return `${prefix}_${timestamp}_${random}`;
};

export {
  checkPermissions,
  generateRandomString,
  generateReferenceId,
  toCamelCase,
  toSnakeCase,
  transformKeys,
};
