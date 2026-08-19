export class A2hError extends Error {
  readonly exitCode: number;

  constructor(message: string, exitCode = 1) {
    super(message);
    this.name = "A2hError";
    this.exitCode = exitCode;
  }
}
