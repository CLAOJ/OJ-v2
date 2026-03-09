import { Comment } from '@/types';

export interface CommentNode extends Comment {
    children: CommentNode[];
}

export function buildCommentTree(comments: Comment[]): CommentNode[] {
    const map: Record<number, CommentNode> = {};
    const roots: CommentNode[] = [];

    comments.forEach(c => {
        map[c.id] = { ...c, children: [] };
    });

    comments.forEach(c => {
        if (c.parent_id && map[c.parent_id]) {
            map[c.parent_id].children.push(map[c.id]);
        } else {
            roots.push(map[c.id]);
        }
    });

    return roots;
}
