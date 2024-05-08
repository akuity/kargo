import { AnyNodeType } from '../types';

export class IndexCache {
  indeces: { [key: string]: number } = {};
  predicate: (node: AnyNodeType, key: string) => boolean;

  constructor(predicate: (node: AnyNodeType, key: string) => boolean) {
    this.predicate = predicate;
  }

  get(key: string, nodes: AnyNodeType[]) {
    let index = this.indeces[key];
    if (index === undefined) {
      index = nodes.findIndex((node) => this.predicate(node, key));
      this.indeces[key] = index;
    }
    return index;
  }
}
