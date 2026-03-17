/**
 * SettingsPage Component
 * Comprehensive user profile and preference settings
 *
 * Features:
 * - Profile info management (name, bio, avatar)
 * - Privacy settings (location consent)
 * - Notification preferences
 * - Linked accounts (Stripe, artist profile)
 * - Session management
 * - Account deletion
 */

import React, { useState, useRef, useCallback } from 'react';
import { DarkModeToggle } from '../components/DarkModeToggle';
import { NotificationSettings } from '../components/NotificationSettings';
import { Avatar } from '../components/Avatar';
import { Input } from '../components/ui/Input';
import { Button } from '../components/ui/Button';
import { ConfirmModal } from '../components/ui/Modal';
import { useTheme } from '../stores/themeStore';
import { useAuth } from '../stores/authStore';
import { useToasts } from '../stores/toastStore';
import { useFileValidation, FileValidationPresets } from '../hooks/useFileValidation';
// TODO: Uncomment when API endpoints are implemented
// import { apiClient } from '../lib/api-client';

interface UserProfile {
  name: string;
  bio: string;
  avatarUrl?: string;
  allowPreciseLocation: boolean;
  stripeConnected: boolean;
  artistProfileLinked: boolean;
}

export const SettingsPage: React.FC = () => {
  const theme = useTheme();
  const { user, logout } = useAuth();
  const toast = useToasts();

  // File validation for avatars
  const validateAvatar = useFileValidation(FileValidationPresets.images);

  // Profile state
  const [profile, setProfile] = useState<UserProfile>({
    name: user?.did.split(':').pop()?.substring(0, 8) || 'User',
    bio: '',
    avatarUrl: undefined,
    allowPreciseLocation: false,
    stripeConnected: false,
    artistProfileLinked: false,
  });

  const [isProfileSaving, setIsProfileSaving] = useState(false);
  const [profileSaveSuccess, setProfileSaveSuccess] = useState(false);
  const [profileError, setProfileError] = useState<string | null>(null);

  // Avatar upload state
  const [isUploadingAvatar, setIsUploadingAvatar] = useState(false);
  const [avatarPreview, setAvatarPreview] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  // Delete account state
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [isDeletingAccount, setIsDeletingAccount] = useState(false);

  // Session management state
  const [isLoggingOutOthers, setIsLoggingOutOthers] = useState(false);

  // Handle profile form changes
  const handleProfileChange = (field: keyof UserProfile, value: string | boolean) => {
    setProfile((prev) => ({ ...prev, [field]: value }));
    setProfileSaveSuccess(false);
    setProfileError(null);
  };

  // Handle avatar file selection
  const handleAvatarSelect = useCallback(
    async (event: React.ChangeEvent<HTMLInputElement>) => {
      const file = event.target.files?.[0];
      if (!file) return;

      // Validate file using validation hook
      const validation = validateAvatar(file);
      if (!validation.isValid) {
        setProfileError(validation.error);
        return;
      }

      // Create preview and wait for it to complete
      try {
        const previewUrl = await new Promise<string>((resolve, reject) => {
          const reader = new FileReader();
          reader.onloadend = () => resolve(reader.result as string);
          reader.onerror = reject;
          reader.readAsDataURL(file);
        });

        setAvatarPreview(previewUrl);

        // Upload avatar with the preview URL
        await handleAvatarUpload(file, previewUrl);
      } catch (error) {
        console.error('Failed to read file:', error);
        setProfileError('Failed to read image file');
      }
    },
    [validateAvatar]
  );

  // Upload avatar to backend
  const handleAvatarUpload = async (file: File, previewUrl: string) => {
    setIsUploadingAvatar(true);
    setProfileError(null);

    try {
      const formData = new FormData();
      formData.append('avatar', file);

      // TODO: Implement actual avatar upload API
      // const response = await apiClient.post<{ url: string }>('/users/avatar', formData);

      // Simulate upload
      await new Promise((resolve) => setTimeout(resolve, 1000));

      // Update profile with new avatar URL
      setProfile((prev) => ({ ...prev, avatarUrl: previewUrl }));
      setProfileSaveSuccess(true);
    } catch (error) {
      console.error('Avatar upload failed:', error);
      setProfileError('Failed to upload avatar. Please try again.');
      setAvatarPreview(null);
    } finally {
      setIsUploadingAvatar(false);
    }
  };

  // Save profile changes
  const handleSaveProfile = async () => {
    setIsProfileSaving(true);
    setProfileError(null);
    setProfileSaveSuccess(false);

    try {
      // TODO: Implement actual profile update API
      // await apiClient.patch('/users/profile', {
      //   name: profile.name,
      //   bio: profile.bio,
      //   allow_precise_location: profile.allowPreciseLocation,
      // });

      // Simulate API call
      await new Promise((resolve) => setTimeout(resolve, 500));

      setProfileSaveSuccess(true);

      // Clear success message after 3 seconds
      setTimeout(() => setProfileSaveSuccess(false), 3000);
    } catch (error) {
      console.error('Profile save failed:', error);
      setProfileError('Failed to save profile. Please try again.');
    } finally {
      setIsProfileSaving(false);
    }
  };

  // Handle logout other devices
  const handleLogoutOtherDevices = async () => {
    setIsLoggingOutOthers(true);

    try {
      // TODO: Implement actual session management API
      // await apiClient.post('/auth/logout-other-devices');

      // Simulate API call
      await new Promise((resolve) => setTimeout(resolve, 1000));

      toast.success('Successfully logged out of all other devices');
    } catch (error) {
      console.error('Logout other devices failed:', error);
      toast.error('Failed to logout other devices. Please try again.');
    } finally {
      setIsLoggingOutOthers(false);
    }
  };

  // Handle account deletion
  const handleDeleteAccount = async () => {
    setIsDeletingAccount(true);

    try {
      // TODO: Implement actual account deletion API
      // await apiClient.delete('/users/account');

      // Simulate API call
      await new Promise((resolve) => setTimeout(resolve, 1000));

      // Logout user
      logout();

      // Redirect to home
      window.location.href = '/';
    } catch (error) {
      console.error('Account deletion failed:', error);
      toast.error('Failed to delete account. Please try again.');
      setIsDeletingAccount(false);
      setShowDeleteConfirm(false);
    }
  };

  return (
    <div className="min-h-screen bg-background text-foreground p-4 md:p-8 theme-transition">
      <div className="max-w-4xl mx-auto">
        <h1 className="text-3xl md:text-4xl font-bold mb-8 text-foreground">Settings</h1>

        <div className="space-y-6">
          {/* Profile Section */}
          <section className="bg-background-secondary border border-border rounded-none p-6 theme-transition">
            <h2 className="text-2xl font-semibold mb-4 text-foreground">Profile</h2>

            {/* Avatar Upload */}
            <div className="flex items-start gap-6 mb-6 pb-6 border-b border-border">
              <div className="flex-shrink-0">
                <Avatar src={avatarPreview || profile.avatarUrl} name={profile.name} size="xl" />
              </div>

              <div className="flex-1">
                <h3 className="text-lg font-medium text-foreground mb-2">Profile Picture</h3>
                <p className="text-sm text-foreground-secondary mb-3">
                  Upload a photo to personalize your profile. JPG, PNG or WebP, max 5MB.
                </p>
                <input
                  ref={fileInputRef}
                  type="file"
                  accept="image/*"
                  onChange={handleAvatarSelect}
                  className="hidden"
                  aria-label="Upload avatar"
                />
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={() => fileInputRef.current?.click()}
                  isLoading={isUploadingAvatar}
                >
                  {isUploadingAvatar ? 'Uploading...' : 'Change Avatar'}
                </Button>
              </div>
            </div>

            {/* Profile Form */}
            <div className="space-y-4">
              <Input
                label="Display Name"
                value={profile.name}
                onChange={(e) => handleProfileChange('name', e.target.value)}
                placeholder="Enter your display name"
                fullWidth
              />

              <div>
                <label htmlFor="bio" className="block text-sm font-medium text-foreground mb-1.5">
                  Bio
                </label>
                <textarea
                  id="bio"
                  value={profile.bio}
                  onChange={(e) => handleProfileChange('bio', e.target.value)}
                  placeholder="Tell us about yourself and your music interests..."
                  rows={4}
                  className="w-full px-3 py-2 rounded-none bg-background border border-border text-foreground placeholder:text-foreground-muted transition-none theme-transition focus:outline-none focus-visible:ring-2 focus-visible:ring-offset-1 focus-visible:ring-brand-primary focus:border-brand-primary"
                />
              </div>

              {profileError && (
                <p className="text-sm text-status-error" role="alert">
                  {profileError}
                </p>
              )}

              {profileSaveSuccess && (
                <p className="text-sm text-status-success" role="status">
                  Profile saved successfully!
                </p>
              )}

              <Button variant="primary" onClick={handleSaveProfile} isLoading={isProfileSaving}>
                Save Profile
              </Button>
            </div>
          </section>

          {/* Privacy Settings */}
          <section className="bg-background-secondary border border-border rounded-none p-6 theme-transition">
            <h2 className="text-2xl font-semibold mb-4 text-foreground">Privacy</h2>

            <div className="space-y-4">
              <div className="flex items-start justify-between py-4 border-b border-border">
                <div className="flex-1">
                  <h3 className="text-lg font-medium text-foreground mb-1">Precise Location</h3>
                  <p className="text-sm text-foreground-secondary">
                    Allow scenes and events to display your precise location. When disabled, only
                    approximate location (within ~5km) is shown for privacy.
                  </p>
                </div>
                <label className="relative inline-flex items-center cursor-pointer ml-4">
                  <input
                    type="checkbox"
                    checked={profile.allowPreciseLocation}
                    onChange={(e) => handleProfileChange('allowPreciseLocation', e.target.checked)}
                    className="sr-only peer"
                    aria-label="Allow precise location"
                  />
                  <div className="w-11 h-6 bg-background border border-border peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-brand-primary rounded-full peer peer-checked:after:translate-x-full rtl:peer-checked:after:-translate-x-full peer-checked:after:border-background after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-foreground after:border-border after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-brand-primary peer-checked:border-brand-primary"></div>
                </label>
              </div>

              <div className="bg-background border border-neon-cyan rounded-none p-4">
                <p className="text-sm text-foreground-secondary">
                  <strong>Privacy First:</strong> Location data is always protected. Even when
                  precise location is enabled, it's only shared with scenes you're a member of and
                  never sold to third parties.
                </p>
              </div>
            </div>
          </section>

          {/* Appearance Section */}
          <section className="bg-background-secondary border border-border rounded-none p-6 theme-transition">
            <h2 className="text-2xl font-semibold mb-4 text-foreground">Appearance</h2>

            <div className="flex items-center justify-between py-4">
              <div>
                <h3 className="text-lg font-medium text-foreground mb-1">Theme</h3>
                <p className="text-sm text-foreground-secondary">
                  Current theme: <span className="font-semibold">{theme}</span>
                </p>
              </div>
              <DarkModeToggle showLabel={true} />
            </div>
          </section>

          {/* Notifications Section */}
          <NotificationSettings />

          {/* Linked Accounts */}
          <section className="bg-background-secondary border border-border rounded-none p-6 theme-transition">
            <h2 className="text-2xl font-semibold mb-4 text-foreground">Linked Accounts</h2>

            <div className="space-y-4">
              {/* Stripe Connect */}
              <div className="flex items-center justify-between py-4 border-b border-border">
                <div>
                  <h3 className="text-lg font-medium text-foreground mb-1">Stripe Connect</h3>
                  <p className="text-sm text-foreground-secondary">
                    {profile.stripeConnected
                      ? 'Connected - Receive direct payments for your events and content'
                      : 'Connect Stripe to receive payments directly from your audience'}
                  </p>
                </div>
                <Button variant={profile.stripeConnected ? 'secondary' : 'primary'} size="sm">
                  {profile.stripeConnected ? 'Manage' : 'Connect'}
                </Button>
              </div>

              {/* Artist Profile */}
              <div className="flex items-center justify-between py-4">
                <div>
                  <h3 className="text-lg font-medium text-foreground mb-1">Artist Profile</h3>
                  <p className="text-sm text-foreground-secondary">
                    {profile.artistProfileLinked
                      ? 'Your artist profile is active and visible to the community'
                      : 'Create an artist profile to showcase your work and events'}
                  </p>
                </div>
                <Button variant={profile.artistProfileLinked ? 'secondary' : 'primary'} size="sm">
                  {profile.artistProfileLinked ? 'Edit' : 'Create'}
                </Button>
              </div>
            </div>
          </section>

          {/* Session Management */}
          <section className="bg-background-secondary border border-border rounded-none p-6 theme-transition">
            <h2 className="text-2xl font-semibold mb-4 text-foreground">Session Management</h2>

            <div className="flex items-start justify-between">
              <div className="flex-1">
                <h3 className="text-lg font-medium text-foreground mb-1">Other Devices</h3>
                <p className="text-sm text-foreground-secondary">
                  Sign out of all other devices where you're currently logged in. Your current
                  session will remain active.
                </p>
              </div>
              <Button
                variant="secondary"
                size="sm"
                onClick={handleLogoutOtherDevices}
                isLoading={isLoggingOutOthers}
                className="ml-4"
              >
                Logout Other Devices
              </Button>
            </div>
          </section>

          {/* Danger Zone */}
          <section className="bg-background-secondary border-2 border-status-error rounded-none p-6 theme-transition">
            <h2 className="text-2xl font-semibold mb-4 text-status-error">Danger Zone</h2>

            <div className="flex items-start justify-between">
              <div className="flex-1">
                <h3 className="text-lg font-medium text-foreground mb-1">Delete Account</h3>
                <p className="text-sm text-foreground-secondary">
                  Permanently delete your account and all associated data. This action cannot be
                  undone. All your scenes, events, posts, and memberships will be deleted.
                </p>
              </div>
              <Button
                variant="danger"
                size="sm"
                onClick={() => setShowDeleteConfirm(true)}
                className="ml-4"
              >
                Delete Account
              </Button>
            </div>
          </section>
        </div>
      </div>

      {/* Delete Account Confirmation Modal */}
      <ConfirmModal
        isOpen={showDeleteConfirm}
        onClose={() => setShowDeleteConfirm(false)}
        onConfirm={handleDeleteAccount}
        title="Delete Account?"
        message="Are you absolutely sure? This will permanently delete your account, including all scenes, events, posts, and memberships. This action cannot be undone."
        confirmText="Yes, Delete My Account"
        cancelText="Cancel"
        variant="danger"
        isLoading={isDeletingAccount}
      />
    </div>
  );
};
