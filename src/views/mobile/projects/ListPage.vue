<template>
    <f7-page ptr @ptr:refresh="reloadByPull" @page:afterin="onPageAfterIn">
        <f7-navbar>
            <f7-nav-left :back-link="tt('Back')"></f7-nav-left>
            <f7-nav-title :title="tt('Projects')"></f7-nav-title>
            <f7-nav-right :class="{ 'navbar-compact-icons': true, 'disabled': loading || updating }">
                <f7-link :icon-f7="showHidden ? 'eye_slash' : 'eye'" :aria-label="tt('Show Hidden Projects')"
                         v-if="hasHiddenProjects" @click="showHidden = !showHidden"></f7-link>
                <f7-link icon-f7="plus" :aria-label="tt('Add Project')" @click="startAdding"></f7-link>
            </f7-nav-right>
        </f7-navbar>

        <f7-block class="margin-top-half margin-bottom-half">
            <small>{{ tt('Group expenses and income from different categories into a project, for example a house renovation, and see what it cost in total.') }}</small>
        </f7-block>

        <f7-list strong inset dividers media-list :class="{ 'skeleton-text': loading }">
            <f7-list-item :title="tt('No available project')" v-if="!loading && !projects.length"></f7-list-item>
            <f7-list-item swipeout
                          :id="getProjectDomId(project)"
                          :key="project.id"
                          :link="`/projects/report?id=${project.id}`"
                          :title="project.name"
                          :after="project.transactionCount ? `${project.transactionCount}` : ''"
                          :text="getProjectDateRangeText(project)"
                          :footer="project.comment"
                          :class="{ 'project-hidden': project.hidden }"
                          v-for="project in projects">
                <template #media>
                    <f7-icon f7="briefcase"></f7-icon>
                </template>
                <f7-swipeout-actions :left="textDirection === TextDirection.RTL"
                                     :right="textDirection === TextDirection.LTR">
                    <f7-swipeout-button color="orange" close :text="tt('Edit')" @click="startEditing(project)"></f7-swipeout-button>
                    <f7-swipeout-button color="blue" close :text="project.hidden ? tt('Show') : tt('Hide')" @click="hide(project, !project.hidden)"></f7-swipeout-button>
                    <f7-swipeout-button color="red" :text="tt('Delete')" @click="remove(project)"></f7-swipeout-button>
                </f7-swipeout-actions>
            </f7-list-item>
        </f7-list>

        <f7-popup push swipe-to-close :opened="!!editingProject" @popup:closed="cancelEditing">
            <f7-page v-if="editingProject">
                <f7-navbar>
                    <f7-nav-left>
                        <f7-link popup-close :text="tt('Cancel')"></f7-link>
                    </f7-nav-left>
                    <f7-nav-title :title="editingProject.id ? tt('Edit Project') : tt('Add Project')"></f7-nav-title>
                    <f7-nav-right>
                        <f7-link :class="{ 'disabled': updating || !editingProject.name.trim() }" :text="tt('Save')" @click="save"></f7-link>
                    </f7-nav-right>
                </f7-navbar>
                <f7-list strong inset dividers class="margin-top">
                    <f7-list-input
                        type="text"
                        autocomplete="off"
                        clear-button
                        :label="tt('Project Name')"
                        :placeholder="tt('Project Name')"
                        v-model:value="editingProject.name"
                    ></f7-list-input>
                    <f7-list-input
                        type="textarea"
                        autocomplete="off"
                        :label="tt('Description')"
                        :placeholder="tt('Your project description (optional)')"
                        v-model:value="editingProject.comment"
                    ></f7-list-input>
                </f7-list>
            </f7-page>
        </f7-popup>
    </f7-page>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue';
import type { Router } from 'framework7/types';

import { useI18n } from '@/locales/helpers.ts';
import { useI18nUIComponents, showLoading, hideLoading } from '@/lib/ui/mobile.ts';
import { useProjectListPageBase } from '@/views/base/projects/ProjectPageBase.ts';

import { TextDirection } from '@/core/text.ts';

import type { ProjectInfoResponse } from '@/models/project.ts';

const props = defineProps<{
    f7router: Router.Router;
}>();

const { tt, getCurrentLanguageTextDirection } = useI18n();
const { showToast, showConfirm, routeBackOnError } = useI18nUIComponents();
const {
    loading,
    updating,
    showHidden,
    editingProject,
    projects,
    hasHiddenProjects,
    reload,
    startAdding,
    startEditing,
    cancelEditing,
    saveEditingProject,
    hideProject,
    deleteProject,
    getProjectDateRangeText
} = useProjectListPageBase();

const textDirection = computed<TextDirection>(() => getCurrentLanguageTextDirection());
const loadingError = ref<unknown | null>(null);

function getProjectDomId(project: ProjectInfoResponse): string {
    return 'project_' + project.id;
}

function showError(error: unknown): void {
    const pageError = error as { processed?: boolean, message?: string };

    if (!pageError.processed) {
        showToast(pageError.message || error as string);
    }
}

function reloadByPull(done?: () => void): void {
    reload(true).then(() => {
        done?.();
    }).catch(error => {
        done?.();
        showError(error);
    });
}

function save(): void {
    const isNew = !editingProject.value?.id;
    showLoading();

    saveEditingProject().then(project => {
        hideLoading();

        if (project) {
            showToast(isNew ? 'You have added a new project' : 'You have saved this project');
        }
    }).catch(error => {
        hideLoading();
        showError(error);
    });
}

function hide(project: ProjectInfoResponse, hidden: boolean): void {
    showLoading();

    hideProject(project, hidden).then(() => {
        hideLoading();
    }).catch(error => {
        hideLoading();
        showError(error);
    });
}

function remove(project: ProjectInfoResponse): void {
    showConfirm('Are you sure you want to delete this project? Its transactions will be kept but no longer belong to it.', () => {
        showLoading();

        deleteProject(project).then(() => {
            hideLoading();
        }).catch(error => {
            hideLoading();
            showError(error);
        });
    });
}

function onPageAfterIn(): void {
    routeBackOnError(props.f7router, loadingError);
}

reload(true).catch(error => {
    if (error.processed) {
        return;
    }

    loadingError.value = error;
    showToast(error.message || error);
});
</script>

<style>
.project-hidden .item-title {
    opacity: 0.6;
}
</style>
