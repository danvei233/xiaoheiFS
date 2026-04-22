import { AppRouteRecord } from '@/types/router'

export const cmsRoutes: AppRouteRecord = {
  path: '/cms',
  name: 'Cms',
  component: '/index/index',
  meta: {
    title: 'CMS',
    icon: 'ri:article-line',
    roles: ['R_SUPER', 'R_ADMIN']
  },
  children: [
    {
      path: 'categories',
      name: 'CmsCategories',
      component: '/cms/categories',
      meta: {
        title: '文章分类',
        keepAlive: true,
        roles: ['R_SUPER', 'R_ADMIN'],
        authList: [
          { title: 'List Categories', authMark: 'cms_category.list' },
          { title: 'Create Categories', authMark: 'cms_category.create' },
          { title: 'Update Categories', authMark: 'cms_category.update' },
          { title: 'Delete Categories', authMark: 'cms_category.delete' }
        ]
      }
    },
    {
      path: 'posts',
      name: 'CmsPosts',
      component: '/cms/posts',
      meta: {
        title: '文章内容',
        keepAlive: true,
        roles: ['R_SUPER', 'R_ADMIN'],
        authList: [
          { title: 'List Posts', authMark: 'cms_post.list' },
          { title: 'Create Posts', authMark: 'cms_post.create' },
          { title: 'Update Posts', authMark: 'cms_post.update' },
          { title: 'Delete Posts', authMark: 'cms_post.delete' }
        ]
      }
    },
    {
      path: 'blocks',
      name: 'CmsBlocks',
      component: '/cms/blocks',
      meta: {
        title: '页面模块',
        keepAlive: true,
        roles: ['R_SUPER', 'R_ADMIN'],
        authList: [
          { title: 'List Blocks', authMark: 'cms_block.list' },
          { title: 'Create Blocks', authMark: 'cms_block.create' },
          { title: 'Update Blocks', authMark: 'cms_block.update' },
          { title: 'Delete Blocks', authMark: 'cms_block.delete' }
        ]
      }
    },
    {
      path: 'uploads',
      name: 'CmsUploads',
      component: '/cms/uploads',
      meta: {
        title: '媒体资源',
        keepAlive: true,
        roles: ['R_SUPER', 'R_ADMIN'],
        authList: [
          { title: 'List Uploads', authMark: 'upload.list' },
          { title: 'Create Uploads', authMark: 'upload.create' }
        ]
      }
    }
  ]
}
